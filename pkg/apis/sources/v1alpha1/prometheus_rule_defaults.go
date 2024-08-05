/*
Copyright 2024 The Knative Authors.

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
	"context"

	"knative.dev/pkg/apis"
)

func (s *PrometheusRuleSource) SetDefaults(ctx context.Context) {
	ctx = apis.WithinParent(ctx, s.ObjectMeta)

	if s.Labels == nil {
		s.Labels = map[string]string{}
	}
	s.Labels[SourceNameLabel] = s.Name

	s.Spec.SetDefaults(ctx)
}

func (s *PrometheusRuleSourceSpec) SetDefaults(ctx context.Context) {
	for i := range s.Rule.Rule.Groups {
		for j := range s.Rule.Rule.Groups[i].Rules {
			if s.Rule.Rule.Groups[i].Rules[j].Labels == nil {
				s.Rule.Rule.Groups[i].Rules[j].Labels = map[string]string{}
			}
			s.Rule.Rule.Groups[i].Rules[j].Labels[SourceNameLabel] = apis.ParentMeta(ctx).Name
			s.Rule.Rule.Groups[i].Rules[j].Labels[SourceNamespaceLabel] = apis.ParentMeta(ctx).Namespace
		}
	}
}
