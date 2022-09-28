package listener

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/cloud/pkg/common/informers"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	routerConfig "github.com/kubeedge/kubeedge/cloud/pkg/router/config"
	"github.com/kubeedge/kubeedge/cloud/pkg/router/utils"
	"github.com/kubeedge/kubeedge/pkg/util"
)

const MaxMessageBytes = 12 * (1 << 20)

var (
	RestHandlerInstance = &RestHandler{}
	routerURL           string
)

type RestHandler struct {
	restTimeout time.Duration
	handlers    sync.Map
	port        int
	bindAddress string

	nodeLister corelisters.NodeLister
}

func InitHandler() {
	timeout := routerConfig.Config.RestTimeout
	if timeout <= 0 {
		timeout = 60
	}
	RestHandlerInstance.restTimeout = time.Duration(timeout) * time.Second
	RestHandlerInstance.bindAddress = routerConfig.Config.Address
	RestHandlerInstance.port = int(routerConfig.Config.Port)
	if RestHandlerInstance.port <= 0 {
		RestHandlerInstance.port = 9443
	}
	RestHandlerInstance.nodeLister = informers.GetInformersManager().GetK8sInformerFactory().Core().V1().Nodes().Lister()

	routerAddr := routerConfig.Config.Address
	if routerAddr == "0.0.0.0" {
		hostnameOverride := util.GetHostname()
		routerAddr, _ = util.GetLocalIP(hostnameOverride)
	}
	routerURL = fmt.Sprintf("http://%s:%d", routerAddr, routerConfig.Config.Port)
	klog.Infof("rest init: %v", RestHandlerInstance)
}

func (rh *RestHandler) Serve() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", rh.httpHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", rh.bindAddress, rh.port),
		Handler: mux,
		// TODO: add tls for router
	}
	klog.Infof("router server listening in %d...", rh.port)
	//err := server.ListenAndServeTLS("", "")
	if err := server.ListenAndServe(); err != nil {
		klog.Errorf("start rest endpoint failed, err: %v", err)
	}
}

func (rh *RestHandler) AddListener(key interface{}, han Handle) {
	path, ok := key.(string)
	if !ok {
		return
	}

	rh.handlers.Store(path, han)
}

func (rh *RestHandler) RemoveListener(key interface{}) {
	path, ok := key.(string)
	if !ok {
		return
	}
	rh.handlers.Delete(path)
}

func (rh *RestHandler) matchedPath(uri string) (string, bool) {
	var candidateRes string
	rh.handlers.Range(func(key, value interface{}) bool {
		pathReg := key.(string)
		if match := utils.IsMatch(pathReg, uri); match {
			if candidateRes != "" && utils.RuleContains(pathReg, candidateRes) {
				return true
			}
			candidateRes = pathReg
		}
		return true
	})
	if candidateRes == "" {
		return "", false
	}
	return candidateRes, true
}

func (rh *RestHandler) httpHandler(w http.ResponseWriter, r *http.Request) {
	uriSections := strings.Split(r.RequestURI, "/")
	if len(uriSections) < 2 {
		// URL format incorrect
		klog.Warningf("url format incorrect: %s", r.URL.String())
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("Request error")); err != nil {
			klog.Errorf("Response write error: %s, %s", r.RequestURI, err.Error())
		}
		return
	}

	matchPath, exist := rh.matchedPath(r.RequestURI)
	if !exist {
		klog.Warningf("URL format incorrect: %s", r.RequestURI)
		w.WriteHeader(http.StatusNotFound)
		if _, err := w.Write([]byte("Request error")); err != nil {
			klog.Errorf("Response write error: %s, %s", r.RequestURI, err.Error())
		}
		return
	}

	node, err := rh.nodeLister.Get(uriSections[1])
	if err != nil || node.Annotations == nil || node.Annotations[modules.RouterURLAnnotationKey] == "" {
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("No rule match"))
		klog.Infof("no rule match, write result: %v", err)
		return
	}

	if node.Annotations[modules.RouterURLAnnotationKey] != routerURL {
		serveReverseProxy(node.Annotations[modules.RouterURLAnnotationKey], w, r)
		return
	}

	v, ok := rh.handlers.Load(matchPath)
	if !ok {
		klog.Warningf("No matched handler for path: %s", matchPath)
		return
	}
	handle, ok := v.(Handle)
	if !ok {
		klog.Errorf("invalid convert to Handle. match path: %s", matchPath)
		return
	}
	aReaderCloser := http.MaxBytesReader(w, r.Body, MaxMessageBytes)
	b, err := ioutil.ReadAll(aReaderCloser)
	if err != nil {
		klog.Errorf("request error, write result: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		if _, err = w.Write([]byte("Request error,body is null")); err != nil {
			klog.Errorf("Response write error: %s, %s", r.RequestURI, err.Error())
		}
		return
	}

	params := make(map[string]interface{})
	msgID := uuid.New().String()
	params["messageID"] = msgID
	params["request"] = r
	params["timeout"] = rh.restTimeout
	params["data"] = b

	resp, err := handle(params)
	if err != nil {
		klog.Errorf("handle request error, msg id: %s, err: %v", msgID, err)
		return
	}
	response, ok := resp.(*http.Response)
	if !ok {
		klog.Errorf("response convert error, msg id: %s, reason: %v", msgID, err)
		return
	}
	body, err := ioutil.ReadAll(io.LimitReader(response.Body, MaxMessageBytes))
	if err != nil {
		klog.Errorf("response body read error, msg id: %s, reason: %v", msgID, err)
		return
	}
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(response.StatusCode)
	if _, err = w.Write(body); err != nil {
		klog.Errorf("response body write error, msg id: %s, reason: %v", msgID, err)
		return
	}
	klog.Infof("response to client, msg id: %s, write result: success", msgID)
}

func (rh *RestHandler) IsMatch(key interface{}, message interface{}) bool {
	res, ok := key.(string)
	if !ok {
		return false
	}
	uri, ok := message.(string)
	if !ok {
		return false
	}
	return utils.IsMatch(res, uri)
}

// Serve a reverse proxy for a given url
func serveReverseProxy(target string, res http.ResponseWriter, req *http.Request) {
	// parse the url
	url, _ := url.Parse(target)

	// create the reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(url)

	// Update the headers to allow for SSL redirection
	req.URL.Host = url.Host
	req.URL.Scheme = url.Scheme
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = url.Host

	// Note that ServeHttp is non blocking and uses a go routine under the hood
	proxy.ServeHTTP(res, req)
}
