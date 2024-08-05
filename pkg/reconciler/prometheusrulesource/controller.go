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

package prometheusrulesource

import (
	"context"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/eventing/pkg/apis/feature"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/resolver"

	sources "knative.dev/eventing-prometheus/pkg/apis/sources/v1alpha1"
	prometheusruleinformer "knative.dev/eventing-prometheus/pkg/client/injection/informers/sources/v1alpha1/prometheusrulesource"
	"knative.dev/eventing-prometheus/pkg/client/injection/reconciler/sources/v1alpha1/prometheusrulesource"
	promclient "knative.dev/eventing-prometheus/pkg/client/prometheus/injection/client"
	promruleinformer "knative.dev/eventing-prometheus/pkg/client/prometheus/injection/informers/monitoring/v1/prometheusrule/filtered"
	alertmanagerconfiginformer "knative.dev/eventing-prometheus/pkg/client/prometheus/injection/informers/monitoring/v1alpha1/alertmanagerconfig/filtered"
)

// NewController initializes the controller and is called by the generated code
// Registers event handlers to enqueue events
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	prometheusRuleSourceInformer := prometheusruleinformer.Get(ctx)

	promRuleInformer := promruleinformer.Get(ctx, sources.SourceSelector)
	amcInformer := alertmanagerconfiginformer.Get(ctx, sources.SourceSelector)

	logger := logging.FromContext(ctx)

	store := feature.NewStore(logger)
	store.WatchConfigs(cmw)

	r := &Reconciler{
		resolver:                 nil,
		promClient:               promclient.Get(ctx),
		promRuleLister:           promRuleInformer.Lister(),
		alertManagerConfigLister: amcInformer.Lister(),
	}
	impl := prometheusrulesource.NewImpl(ctx, r, func(impl *controller.Impl) controller.Options {
		return controller.Options{
			ConfigStore: store,
		}
	})
	r.resolver = resolver.NewURIResolverFromTracker(ctx, impl.Tracker)

	prometheusRuleSourceInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))

	promRuleInformer.Informer().AddEventHandler(controller.HandleAll(func(obj interface{}) {
		r, ok := obj.(*monitoringv1.PrometheusRule)
		if !ok {
			return
		}
		name, ok := r.Labels[sources.SourceNameLabel]
		if !ok {
			return
		}
		impl.EnqueueKey(types.NamespacedName{Namespace: r.Namespace, Name: name})
	}))
	amcInformer.Informer().AddEventHandler(controller.HandleAll(func(obj interface{}) {
		r, ok := obj.(*monitoringv1alpha1.AlertmanagerConfig)
		if !ok {
			return
		}
		name, ok := r.Labels[sources.SourceNameLabel]
		if !ok {
			return
		}
		impl.EnqueueKey(types.NamespacedName{Namespace: r.Namespace, Name: name})
	}))

	return impl
}
