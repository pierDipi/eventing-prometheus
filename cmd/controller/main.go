/*
Copyright 2019 The Knative Authors

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

package main

import (
	"knative.dev/pkg/signals"

	sources "knative.dev/eventing-prometheus/pkg/apis/sources/v1alpha1"
	"knative.dev/eventing-prometheus/pkg/reconciler/prometheusrulesource"
	"knative.dev/eventing-prometheus/pkg/reconciler/prometheussource"

	"knative.dev/pkg/injection/sharedmain"

	prometheusinformerfilteredfactory "knative.dev/eventing-prometheus/pkg/client/prometheus/injection/informers/factory/filtered"
)

func main() {
	ctx := signals.NewContext()

	ctx = prometheusinformerfilteredfactory.WithSelectors(ctx, sources.SourceSelector)

	sharedmain.MainWithContext(ctx, "prometheussource-controller",
		prometheussource.NewController,
		prometheusrulesource.NewController,
	)
}
