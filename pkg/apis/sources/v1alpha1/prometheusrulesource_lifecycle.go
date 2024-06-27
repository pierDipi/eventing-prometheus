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
	"knative.dev/pkg/apis"
)

const (
	// PrometheusRuleConditionSinkProvided has status True when the PrometheusRuleSource has been configured with a sink target.
	PrometheusRuleConditionSinkProvided apis.ConditionType = "SinkProvided"
)

var PrometheusRuleCondSet = apis.NewLivingConditionSet(
	PrometheusRuleConditionSinkProvided,
)

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*PrometheusRuleSource) GetConditionSet() apis.ConditionSet {
	return PrometheusRuleCondSet
}

// GetConditionSet retrieves the condition set for this resource. Implements the KRShaped interface.
func (*PrometheusRuleSourceStatus) GetConditionSet() apis.ConditionSet {
	return PrometheusRuleCondSet
}

// GetCondition returns the condition currently associated with the given type, or nil.
func (s *PrometheusRuleSourceStatus) GetCondition(t apis.ConditionType) *apis.Condition {
	return s.GetConditionSet().Manage(s).GetCondition(t)
}

// InitializeConditions sets relevant unset conditions to Unknown state.
func (s *PrometheusRuleSourceStatus) InitializeConditions() {
	s.GetConditionSet().Manage(s).InitializeConditions()
}

// MarkSink sets the condition that the source has a sink configured.
func (s *PrometheusRuleSourceStatus) MarkSink(uri *apis.URL) {
	s.SinkURI = uri
	if !uri.IsEmpty() {
		s.GetConditionSet().Manage(s).MarkTrue(PrometheusRuleConditionSinkProvided)
	} else {
		s.GetConditionSet().Manage(s).MarkUnknown(PrometheusRuleConditionSinkProvided, "SinkEmpty", "Sink has resolved to empty.%s", "")
	}
}

// MarkNoSink sets the condition that the source does not have a sink configured.
func (s *PrometheusRuleSourceStatus) MarkNoSink(reason, messageFormat string, messageA ...interface{}) {
	s.GetConditionSet().Manage(s).MarkFalse(PrometheusRuleConditionSinkProvided, reason, messageFormat, messageA...)
}

// IsReady returns true if the resource is ready overall.
func (s *PrometheusRuleSourceStatus) IsReady() bool {
	return s.GetConditionSet().Manage(s).IsHappy()
}
