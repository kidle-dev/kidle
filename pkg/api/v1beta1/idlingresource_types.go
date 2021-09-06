/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"github.com/kidle-dev/kidle/pkg/utils/array"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	IdlingResources = "idlingresources"

	// IdlingResourceFinalizerName is the name of the idlingresource finalizer
	IdlingResourceFinalizerName = "idlingresource.finalizers.kidle.kidle.dev"

	// TODO
	MetadataIdlingResourceReference = "kidle.kidle.dev/idling-resource-reference"

	// TODO
	MetadataPreviousReplicas = "kidle.kidle.dev/previous-replicas"

	// TODO
	MetadataExpectedState = "kidle.kidle.dev/expected-state"
)

// IdlingResourceSpec defines the desired state of IdlingResource
type IdlingResourceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The reference to the idle-able resource
	IdlingResourceRef CrossVersionObjectReference `json:"idlingResourceRef"`

	// The desired state of idling. Defaults to false.
	// +kubebuilder:default:false
	Idle bool `json:"idle"`

	// +optional
	IdlingStrategy *IdlingStrategy `json:"idlingStrategy,omitempty"`

	// +optional
	WakeupStrategy *WakeupStrategy `json:"wakeupStrategy,omitempty"`
}

// CrossVersionObjectReference contains enough information to let you identify the referred resource.
type CrossVersionObjectReference struct {
	// Kind of the referent; More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds"
	Kind string `json:"kind"`

	// Name of the referent; More info: http://kubernetes.io/docs/user-guide/identifiers#names
	Name string `json:"name"`

	// API version of the referent
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

type IdlingStrategy struct {
	// +optional
	CronStrategy *CronStrategy `json:"cronStrategy,omitempty"`

	// +optional
	InactiveStrategy *InactiveStrategy `json:"inactiveStrategy,omitempty"`
}

type CronStrategy struct {
	// The schedule in Cron format, see https://en.wikipedia.org/wiki/Cron.
	Schedule string `json:"schedule"`
}

type InactiveStrategy struct {
}

type WakeupStrategy struct {
	// +optional
	CronStrategy *CronStrategy `json:"cronStrategy,omitempty"`

	// +optional
	OnCallStrategy *OnCallStrategy `json:"onCallStrategy,omitempty"`
}

type OnCallStrategy struct {
}

// IdlingResourceStatus defines the observed state of IdlingResource
type IdlingResourceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:resource:shortName=ir
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Idle",type="boolean",JSONPath=".spec.idle"
// +kubebuilder:printcolumn:name="RefKind",type="string",JSONPath=".spec.idlingResourceRef.kind"
// +kubebuilder:printcolumn:name="RefName",type="string",JSONPath=".spec.idlingResourceRef.name"

// IdlingResource is the Schema for the idlingresources API
type IdlingResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IdlingResourceSpec   `json:"spec,omitempty"`
	Status IdlingResourceStatus `json:"status,omitempty"`
}

// IsBeingDeleted returns true if a deletion timestamp is set
func (ss *IdlingResource) IsBeingDeleted() bool {
	return !ss.ObjectMeta.DeletionTimestamp.IsZero()
}

// HasFinalizer returns true if the item has the specified finalizer
func (ss *IdlingResource) HasFinalizer(finalizerName string) bool {
	return array.ContainsString(ss.Finalizers, finalizerName)
}

// AddFinalizer adds the specified finalizer
func (ss *IdlingResource) AddFinalizer(finalizerName string) {
	ss.ObjectMeta.Finalizers = append(ss.ObjectMeta.Finalizers, finalizerName)
}

// RemoveFinalizer removes the specified finalizer
func (ss *IdlingResource) RemoveFinalizer(finalizerName string) {
	ss.ObjectMeta.Finalizers = array.RemoveString(ss.ObjectMeta.Finalizers, finalizerName)
}

// +kubebuilder:object:root=true

// IdlingResourceList contains a list of IdlingResource
type IdlingResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IdlingResource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IdlingResource{}, &IdlingResourceList{})
}
