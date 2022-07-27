package metrics

import (
	"time"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	cloudcoreHandleReceiveMessageDuration = map[string]time.Time{}
	cloudcoreSendMessageDuration          = map[string]time.Time{}
)

var (
	CloudHubReceiveMessageCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudcore_cloudhub_receive_message_count",
			Help: "The count of cloudcore cloudhub receive message from edge.",
		},
		[]string{"node_id", "source", "group", "operation", "resource"},
	)
	CloudHubTransmitMessageCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudcore_cloudhub_transmit_message_count",
			Help: "The count of cloudcore cloudhub transmit message to edge.",
		},
		[]string{"node_id", "source", "group", "operation", "resource"},
	)
	CloudcoreMessageHandleDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cloudcore_message_handle_duration_seconds_bucket",
			Help:    "Time for cloudcore handle message distribution in seconds for each source, group, operation and resource.",
			Buckets: []float64{0.1, 0.2, 0.4, 0.8, 1.6, 3.0, 5, 10, 30, 60},
		},
		[]string{"source", "group", "operation", "resource"},
	)
	CloudcoreMessageSendDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "cloudcore_message_send_duration_seconds_bucket",
			Help:    "Time for cloudcore send message distribution in seconds for each source, group, operation and resource.",
			Buckets: []float64{0.1, 0.2, 0.4, 0.8, 1.6, 3.0, 5, 10, 30, 60},
		},
		[]string{"source", "group", "operation", "resource"},
	)
	CloudHubNodeQueue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cloudcore_cloudhub_node_queue_length",
			Help: "The size of cloudcore cloudhub node queue",
		},
		[]string{"node_id"},
	)
	CloudHubNodeListQueue = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cloudcore_cloudhub_node_list_queue_length",
			Help: "The size of cloudcore cloudhub node list queue",
		},
		[]string{"node_id"},
	)
)

func init() {
	prometheus.MustRegister(CloudHubReceiveMessageCount)
	prometheus.MustRegister(CloudHubTransmitMessageCount)
	prometheus.MustRegister(CloudcoreMessageHandleDurationSeconds)
	prometheus.MustRegister(CloudcoreMessageSendDurationSeconds)
	prometheus.MustRegister(CloudHubNodeQueue)
	prometheus.MustRegister(CloudHubNodeListQueue)
}

func RecordCloudHubReceiveMessage(nodeID string, msg *model.Message) {
	go CloudHubReceiveMessageCount.WithLabelValues(
		nodeID,
		msg.Router.Source,
		msg.Router.Group,
		msg.Router.Operation,
		msg.Router.Resource,
	).Inc()
}

func RecordCloudHubTransmitMessage(nodeID string, msg *model.Message) {
	go CloudHubTransmitMessageCount.WithLabelValues(
		nodeID,
		msg.Router.Source,
		msg.Router.Group,
		msg.Router.Operation,
		msg.Router.Resource,
	).Inc()
}

func RecordCloudHubReceiveMessageTime(msg *model.Message) {
	cloudcoreHandleReceiveMessageDuration[msg.GetID()] = time.Now()
}

func RecordCloudcoreMessageHandleDuration(msg *model.Message) {
	go func() {
		uid := msg.GetID()
		start, ok := cloudcoreHandleReceiveMessageDuration[uid]
		if !ok {
			return
		}
		CloudcoreMessageHandleDurationSeconds.WithLabelValues(
			msg.Router.Source,
			msg.Router.Group,
			msg.Router.Operation,
			msg.Router.Resource,
		).Observe(time.Since(start).Seconds())
		delete(cloudcoreHandleReceiveMessageDuration, uid)
	}()
}

func RecordCloudHubBuildMessageTime(msg *model.Message) {
	cloudcoreSendMessageDuration[msg.GetID()] = time.Now()
}

func RecordCloudcoreSendMessageDuration(msg *model.Message) {
	go func() {
		uid := msg.GetID()
		start, ok := cloudcoreSendMessageDuration[uid]
		if !ok {
			return
		}
		CloudcoreMessageSendDurationSeconds.WithLabelValues(
			msg.Router.Source,
			msg.Router.Group,
			msg.Router.Operation,
			msg.Router.Resource,
		).Observe(time.Since(start).Seconds())
		delete(cloudcoreSendMessageDuration, uid)
	}()
}

func RecordCloudHubNodeQueue(nodeID string, size int) {
	go CloudHubNodeQueue.WithLabelValues(nodeID).Set(float64(size))
}

func RecordCloudHubNodeListQueue(nodeID string, size int) {
	go CloudHubNodeListQueue.WithLabelValues(nodeID).Set(float64(size))
}
