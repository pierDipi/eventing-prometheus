/*
Copyright 2024 The Knative Authors

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

package v1alpha1

import (
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis/duck"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/webhook/resourcesemantics"
)

const (
	SourceSelector       = "prometheus.sources.knative.dev"
	SourceNameLabel      = "prometheus.sources.knative.dev/name"
	SourceNamespaceLabel = "prometheus.sources.knative.dev/namespace"
)

// +genclient
// +genreconciler
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true

// PrometheusRuleSource is the Schema for the prometheus rule source API
// +k8s:openapi-gen=true
type PrometheusRuleSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PrometheusRuleSourceSpec   `json:"spec,omitempty"`
	Status PrometheusRuleSourceStatus `json:"status,omitempty"`
}

var _ resourcesemantics.GenericCRD = (*PrometheusRuleSource)(nil)

// Check that Prometheus source can be validated and can be defaulted.
var _ runtime.Object = (*PrometheusRuleSource)(nil)

// Check that we can create OwnerReferences to a PrometheusSource.
var _ kmeta.OwnerRefable = (*PrometheusRuleSource)(nil)

// Check that the type conforms to the duck Knative Resource shape.
var _ duckv1.KRShaped = (*PrometheusRuleSource)(nil)

// Check that PrometheusSource implements the Conditions duck type.
var _ = duck.VerifyType(&PrometheusRuleSource{}, &duckv1.Conditions{})

// PrometheusRuleSourceSpec defines the desired state of PrometheusSource
type PrometheusRuleSourceSpec struct {
	// Rule is the PrometheusRule spec
	Rule PrometheusRuleSpec `json:",inline"`

	// PrometheusRuleEventSpec is the event emitted by PrometheusRuleSource when an alert is firing.
	Event PrometheusRuleEventSpec `json:"event"`

	// SourceSpec inlined spec fields for sources
	duckv1.SourceSpec `json:",inline"`

	// Reply specifies (optionally) how to handle events returned from
	// the Sink target.
	// +optional
	Reply *duckv1.Destination `json:"reply,omitempty"`
}

// PrometheusRuleSpec is a wrapper type for monitoringv1.PrometheusRuleSpec (for validation and
// defaulting)
type PrometheusRuleSpec struct {
	// Rule is the prometheus rule spec.
	Rule monitoringv1.PrometheusRuleSpec `json:"rule"`
	// How long to wait before sending the initial notification.
	// Must match the regular expression`^(([0-9]+)y)?(([0-9]+)w)?(([0-9]+)d)?(([0-9]+)h)?(([0-9]+)m)?(([0-9]+)s)?(([0-9]+)ms)?$`
	// Example: "30s"
	// +optional
	GroupWait string `json:"groupWait,omitempty"`
	// How long to wait before sending an updated notification.
	// Must match the regular expression`^(([0-9]+)y)?(([0-9]+)w)?(([0-9]+)d)?(([0-9]+)h)?(([0-9]+)m)?(([0-9]+)s)?(([0-9]+)ms)?$`
	// Example: "5m"
	// +optional
	GroupInterval string `json:"groupInterval,omitempty"`
	// How long to wait before repeating the last notification.
	// Must match the regular expression`^(([0-9]+)y)?(([0-9]+)w)?(([0-9]+)d)?(([0-9]+)h)?(([0-9]+)m)?(([0-9]+)s)?(([0-9]+)ms)?$`
	// Example: "4h"
	// +optional
	RepeatInterval string `json:"repeatInterval,omitempty"`
	// List of labels to group by.
	// Labels must not be repeated (unique list).
	// Special label "..." (aggregate by all possible labels), if provided, must be the only element in the list.
	// +optional
	GroupBy []string `json:"groupBy,omitempty"`
	// MuteTimeIntervals is a list of TimeInterval names that will mute this route when matched.
	// +optional
	MuteTimeIntervals []string `json:"muteTimeIntervals,omitempty"`
	// ActiveTimeIntervals is a list of TimeInterval names when this route should be active.
	// +optional
	ActiveTimeIntervals []string `json:"activeTimeIntervals,omitempty"`
}

// PrometheusRuleEventSpec is the event emitted by PrometheusRuleSource when an alert is firing.
type PrometheusRuleEventSpec struct {
	Source string `json:"source"`

	Type string `json:"type"`

	Subject string `json:"subject,omitempty"`

	DataSchema string `json:"dataschema,omitempty"`

	Data *apiextensions.JSON `json:"data,omitempty"`
}

// GetGroupVersionKind returns the GroupVersionKind.
func (*PrometheusRuleSource) GetGroupVersionKind() schema.GroupVersionKind {
	return SchemeGroupVersion.WithKind("PrometheusRuleSource")
}

// GetStatus retrieves the duck status for this resource. Implements the KRShaped interface.
func (p *PrometheusRuleSource) GetStatus() *duckv1.Status {
	return &p.Status.Status
}

// PrometheusRuleSourceStatus defines the observed state of PrometheusRuleSource
type PrometheusRuleSourceStatus struct {
	// inherits duck/v1 SourceStatus, which currently provides:
	// * ObservedGeneration - the 'Generation' of the Service that was last
	//   processed by the controller.
	// * Conditions - the latest available observations of a resource's current
	//   state.
	// * SinkURI - the current active sink URI that has been configured for the
	//   Source.
	duckv1.SourceStatus `json:",inline"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PrometheusRuleSourceList contains a list of PrometheusRuleSource
type PrometheusRuleSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PrometheusRuleSource `json:"items"`
}
