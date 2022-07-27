package context

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

var (
	ModuleChannelLength = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cloudcore_beehive_module_channel_length",
			Help: "The length of cloudcore beehive module channel",
		},
		[]string{"module_name"},
	)
)

func init() {
	prometheus.MustRegister(ModuleChannelLength)
}

func StartMetrics() {
	go func() {
		ctx, ok := context.moduleContext.(*ChannelContext)
		if !ok {
			klog.Warning("context.moduleContext assert to ChannelContext error, metrics not start")
			return
		}
		for {
			select {
			case <-Done():
				return
			default:
				for m, c := range ctx.channels {
					ModuleChannelLength.WithLabelValues(m).Set(float64(len(c)))
				}

			}
			time.Sleep(30 * time.Second)
		}
	}()
}
