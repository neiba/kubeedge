package client

import (
	"encoding/json"
	"fmt"

	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/common/message"
	"github.com/kubeedge/kubeedge/edge/pkg/common/modules"
	api "k8s.io/api/core/v1"
)

// EventsGetter has a method to return a EventInterface.
// A group's client should implement this interface.
type EventsGetter interface {
	Events(namespace string) EventsInterface
}

// EventsInterface has methods to work with Event resources.
type EventsInterface interface {
	Create(*api.Event) (*api.Event, error)
	Update(*api.Event) (*api.Event, error)
	Delete(name string) error
	Get(name string) (*api.Event, error)
}

type Events struct {
	namespace string
	send      SendInterface
}

func newEvents(namespace string, s SendInterface) *Events {
	return &Events{
		send:      s,
		namespace: namespace,
	}
}

func (e *Events) Create(event *api.Event) (*api.Event, error) {
	resource := fmt.Sprintf("%s/%s/%s", event.Namespace, model.ResourceTypeEvent, event.Name)
	eventMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.InsertOperation, event)
	msg, err := e.send.SendSync(eventMsg)
	if err != nil {
		return nil, fmt.Errorf("create event failed, err: %v", err)
	}
	content, err := msg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to event failed, err: %v", err)
	}
	return handleEventFromMetaManager(content)
}

func (e *Events) Update(event *api.Event) (*api.Event, error) {
	resource := fmt.Sprintf("%s/%s/%s", event.Namespace, model.ResourceTypeEvent, event.Name)
	eventMsg := message.BuildMsg(modules.MetaGroup, "", modules.EdgedModuleName, resource, model.UpdateOperation, event)
	msg, err := e.send.SendSync(eventMsg)
	if err != nil {
		return nil, fmt.Errorf("create event failed, err: %v", err)
	}
	content, err := msg.GetContentData()
	if err != nil {
		return nil, fmt.Errorf("parse message to event failed, err: %v", err)
	}
	return handleEventFromMetaManager(content)
}

func (e *Events) Delete(name string) error {
	return nil
}

func (e *Events) Get(name string) (*api.Event, error) {
	return nil, nil
}

func handleEventFromMetaManager(content []byte) (*api.Event, error) {
	var event api.Event
	err := json.Unmarshal(content, &event)
	if err != nil {
		return nil, fmt.Errorf("unmarshal message to Event failed, err: %v", err)
	}
	return &event, nil
}
