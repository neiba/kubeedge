package v1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	fakecorev1 "k8s.io/client-go/kubernetes/typed/core/v1/fake"

	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/client"
)

// FakePersistentVolumeClaims implements PersistentVolumeClaimInterface
type FakeEvent struct {
	fakecorev1.FakeEvents
	ns         string
	MetaClient client.CoreInterface
}

// CreateWithEventNamespace makes a new event. Returns the copy of the event the server returns,
// or an error. The namespace to create the event within is deduced from the
// event; it must either match this event client's namespace, or this event
// client must have been created with the "" namespace.
func (e *FakeEvent) CreateWithEventNamespace(event *corev1.Event) (*corev1.Event, error) {
	if e.ns != "" && event.Namespace != e.ns {
		return nil, fmt.Errorf("can't create an event with namespace '%v' in namespace '%v'", event.Namespace, e.ns)
	}
	return e.MetaClient.Events(event.Namespace).Create(event)
}

// UpdateWithEventNamespace modifies an existing event. It returns the copy of the event that the server returns,
// or an error. The namespace and key to update the event within is deduced from the event. The
// namespace must either match this event client's namespace, or this event client must have been
// created with the "" namespace. Update also requires the ResourceVersion to be set in the event
// object.
func (e *FakeEvent) UpdateWithEventNamespace(event *corev1.Event) (*corev1.Event, error) {
	if e.ns != "" && event.Namespace != e.ns {
		return nil, fmt.Errorf("can't create an event with namespace '%v' in namespace '%v'", event.Namespace, e.ns)
	}
	return e.MetaClient.Events(event.Namespace).Update(event)
}

// PatchWithEventNamespace modifies an existing event. It returns the copy of
// the event that the server returns, or an error. The namespace and name of the
// target event is deduced from the incompleteEvent. The namespace must either
// match this event client's namespace, or this event client must have been
// created with the "" namespace.
func (e *FakeEvent) PatchWithEventNamespace(incompleteEvent *corev1.Event, data []byte) (*corev1.Event, error) {
	return e.UpdateWithEventNamespace(incompleteEvent)
}
