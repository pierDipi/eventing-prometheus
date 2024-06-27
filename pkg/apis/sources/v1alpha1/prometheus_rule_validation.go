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

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/google/uuid"
	"knative.dev/pkg/apis"
)

// Validate Prometheus source object fields
func (s *PrometheusRuleSource) Validate(ctx context.Context) *apis.FieldError {
	return s.Spec.Validate(ctx).ViaField("spec")
}

// Validate Prometheus source Spec object fields
func (s *PrometheusRuleSourceSpec) Validate(ctx context.Context) *apis.FieldError {
	var errs *apis.FieldError
	errs = errs.Also(s.SourceSpec.Validate(ctx).ViaField("sink"))
	errs = errs.Also(s.Rule.Validate(ctx).ViaField("rule"))
	errs = errs.Also(s.Event.Validate(ctx).ViaField("event"))
	return errs
}

func (s *PrometheusRuleSpec) Validate(context.Context) *apis.FieldError {
	var errs *apis.FieldError
	if len(s.Rule.Groups) == 0 {
		errs = errs.Also(apis.ErrMissingField("groups"))
	}
	for i, g := range s.Rule.Groups {
		if len(g.Rules) == 0 {
			errs = errs.Also(apis.ErrMissingField("rules")).ViaFieldIndex("groups", i)
		}
	}
	return errs
}

func (s *PrometheusRuleEventSpec) Validate(context.Context) *apis.FieldError {
	var errs *apis.FieldError

	e := cloudevents.NewEvent(cloudevents.VersionV1)
	e.SetID(uuid.New().String())
	e.SetType(s.Type)
	e.SetSource(s.Source)
	e.SetSubject(s.Subject)
	e.SetDataSchema(s.DataSchema)

	e.SetDataContentType(cloudevents.ApplicationJSON)
	e.DataEncoded = s.Data.Raw
	e.DataBase64 = false

	if err := e.Validate(); err != nil {
		errs = errs.Also(apis.ErrInvalidValue(s, "", err.Error()))
	}
	return errs
}
