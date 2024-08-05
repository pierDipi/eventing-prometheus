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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/utils/pointer"
	"k8s.io/utils/ptr"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/resolver"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1alpha1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1alpha1"
	monitoringv1beta1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1beta1"

	sources "knative.dev/eventing-prometheus/pkg/apis/sources/v1alpha1"
	promclient "knative.dev/eventing-prometheus/pkg/client/prometheus/clientset/versioned"
	promlister "knative.dev/eventing-prometheus/pkg/client/prometheus/listers/monitoring/v1"
	alertmanagerconfiglister "knative.dev/eventing-prometheus/pkg/client/prometheus/listers/monitoring/v1alpha1"
)

type Reconciler struct {
	resolver   *resolver.URIResolver
	promClient promclient.Interface

	promRuleLister           promlister.PrometheusRuleLister
	alertManagerConfigLister alertmanagerconfiglister.AlertmanagerConfigLister
}

func (r *Reconciler) ReconcileKind(ctx context.Context, src *sources.PrometheusRuleSource) reconciler.Event {
	if err := r.reconcilePrometheusResources(ctx, src); err != nil {
		return fmt.Errorf("failed to reconcile prometheus resources: %w", err)
	}

	return nil
}

func (r *Reconciler) reconcilePrometheusResources(ctx context.Context, src *sources.PrometheusRuleSource) error {
	if err := r.reconcilePrometheusRule(ctx, src); err != nil {
		return fmt.Errorf("failed to reconcile prometheus rule: %w", err)
	}
	if err := r.reconcileAlertManagerConfig(ctx, src); err != nil {
		return fmt.Errorf("failed to reconcile alert manager config: %w", err)
	}
	return nil
}

func (r *Reconciler) reconcilePrometheusRule(ctx context.Context, src *sources.PrometheusRuleSource) error {
	pr := &monitoringv1.PrometheusRule{
		ObjectMeta: metav1.ObjectMeta{
			Name:        kmeta.ChildName(src.Name, "kne"),
			Namespace:   src.Namespace,
			Labels:      src.Labels,
			Annotations: src.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         src.GetGroupVersionKind().GroupVersion().String(),
					Kind:               src.GetGroupVersionKind().Kind,
					Name:               src.Name,
					UID:                src.UID,
					Controller:         ptr.To(true),
					BlockOwnerDeletion: ptr.To(false),
				},
			},
		},
		Spec: src.Spec.Rule.Rule,
	}
	pr.Labels[sources.SourceSelector] = "true"

	selector := labels.SelectorFromSet(map[string]string{sources.SourceNameLabel: src.Name})
	curr, err := r.promRuleLister.PrometheusRules(src.Namespace).List(selector)
	if err != nil {
		return fmt.Errorf("failed to list prometheus rules: %w", err)
	}
	if len(curr) == 0 {
		if err := r.createPrometheusRule(ctx, src, pr); err != nil {
			return err
		}
		return nil
	}
	toBeDeleted := curr[1:]
	defer r.deletePrometheusRules(ctx, src, toBeDeleted...)

	got := curr[0]
	if equality.Semantic.DeepDerivative(pr, got) {
		return nil
	}

	pr.ObjectMeta.ResourceVersion = got.ObjectMeta.ResourceVersion

	return r.updatePrometheusRule(ctx, src, pr)
}

func (r *Reconciler) reconcileAlertManagerConfig(ctx context.Context, src *sources.PrometheusRuleSource) error {
	addr, err := r.resolver.AddressableFromDestinationV1(ctx, src.Spec.Sink, src)
	if err != nil || addr == nil {
		err := fmt.Errorf("failed to resolve sink address: %w", err)
		src.Status.MarkNoSink("Error", err.Error())
		return err
	}
	src.Status.MarkSink(*addr)

	amc := &monitoringv1alpha1.AlertmanagerConfig{
		ObjectMeta: metav1.ObjectMeta{
			Name:        kmeta.ChildName(src.Name, "kne"),
			Namespace:   src.Namespace,
			Labels:      src.Labels,
			Annotations: src.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         src.GetGroupVersionKind().GroupVersion().String(),
					Kind:               src.GetGroupVersionKind().Kind,
					Name:               src.Name,
					UID:                src.UID,
					Controller:         ptr.To(true),
					BlockOwnerDeletion: ptr.To(false),
				},
			},
		},
		Spec: monitoringv1alpha1.AlertmanagerConfigSpec{
			Route: &monitoringv1alpha1.Route{
				Receiver:       fmt.Sprintf("%s-%s", src.Namespace, src.Name),
				GroupBy:        append([]string{sources.SourceNameLabel, sources.SourceNamespaceLabel}, src.Spec.Rule.GroupBy...),
				GroupWait:      src.Spec.Rule.GroupWait,
				GroupInterval:  src.Spec.Rule.GroupInterval,
				RepeatInterval: src.Spec.Rule.RepeatInterval,
				Matchers: []monitoringv1alpha1.Matcher{
					{
						Name:      sources.SourceNameLabel,
						Value:     src.Name,
						MatchType: monitoringv1alpha1.MatchEqual,
					},
					{
						Name:      sources.SourceNamespaceLabel,
						Value:     src.Name,
						MatchType: monitoringv1alpha1.MatchEqual,
					},
				},
				MuteTimeIntervals:   src.Spec.Rule.MuteTimeIntervals,
				ActiveTimeIntervals: src.Spec.Rule.ActiveTimeIntervals,
			},
			Receivers: []monitoringv1alpha1.Receiver{
				{
					Name: fmt.Sprintf("%s-%s", src.Namespace, src.Name),
					WebhookConfigs: []monitoringv1alpha1.WebhookConfig{
						{
							SendResolved: ptr.To(false),
							URL:          ptr.To(addr.URL.String()),
							URLSecret:    nil,
							HTTPConfig:   nil,
							MaxAlerts:    1,
						},
					},
				},
			},
			InhibitRules: nil,
		},
	}
	amc.Labels[sources.SourceSelector] = "true"

	selector := labels.SelectorFromSet(map[string]string{sources.SourceNameLabel: src.Name})
	curr, err := r.alertManagerConfigLister.AlertmanagerConfigs(src.Namespace).List(selector)
	if err != nil {
		return fmt.Errorf("failed to list alertmanager configs : %w", err)
	}
	if len(curr) == 0 {
		if err := r.createAlertManagerConfig(ctx, src, amc); err != nil {
			return err
		}
		return nil
	}
	toBeDeleted := curr[1:]
	defer r.deleteAlertManagerConfig(ctx, src, toBeDeleted...)

	got := curr[0]
	if equality.Semantic.DeepDerivative(amc, got) {
		return nil
	}

	amc.ObjectMeta.ResourceVersion = got.ObjectMeta.ResourceVersion

	return r.updateAlertManagerConfig(ctx, src, amc)
}

func (r *Reconciler) deleteAlertManagerConfig(ctx context.Context, src *sources.PrometheusRuleSource, toBeDeleted ...*monitoringv1alpha1.AlertmanagerConfig) error {
	for _, amc := range toBeDeleted {
		err := r.promClient.MonitoringV1beta1().
			AlertmanagerConfigs(amc.Namespace).
			Delete(ctx, amc.Name, metav1.DeleteOptions{
				Preconditions: &metav1.Preconditions{
					UID: &amc.UID,
				},
			})
		if err != nil {
			return fmt.Errorf("failed to delete alert manager config %s/%s: %w", amc.Namespace, amc.Name, err)
		}

		controller.GetEventRecorder(ctx).
			Eventf(src, corev1.EventTypeNormal, "Updated", "Updated AlertManager config %s/%s", amc.Namespace, amc.Name)
	}
	return nil
}

func (r *Reconciler) createAlertManagerConfig(ctx context.Context, src *sources.PrometheusRuleSource, amc *monitoringv1alpha1.AlertmanagerConfig) error {
	_, err := r.promClient.MonitoringV1alpha1().
		AlertmanagerConfigs(amc.Namespace).
		Create(ctx, amc, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create alert manager config %s/%s: %w", amc.Namespace, amc.Name, err)
	}

	controller.GetEventRecorder(ctx).
		Eventf(src, corev1.EventTypeNormal, "Created", "Created AlertManager config %s/%s", amc.Namespace, amc.Name)

	return nil
}

func (r *Reconciler) updateAlertManagerConfig(ctx context.Context, src *sources.PrometheusRuleSource, amc *monitoringv1alpha1.AlertmanagerConfig) error {
	_, err := r.promClient.MonitoringV1alpha1().
		AlertmanagerConfigs(amc.Namespace).
		Update(ctx, amc, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update alert manager config %s/%s: %w", amc.Namespace, amc.Name, err)
	}

	controller.GetEventRecorder(ctx).
		Eventf(src, corev1.EventTypeNormal, "Updated", "Updated AlertManager config %s/%s", amc.Namespace, amc.Name)

	return nil
}

func convert(acm *monitoringv1alpha1.AlertmanagerConfig) *monitoringv1beta1.AlertmanagerConfig {
	r := &monitoringv1beta1.AlertmanagerConfig{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: *acm.ObjectMeta.DeepCopy(),
	}

	if acm.Spec.Route != nil {
		r.Spec.Route = &monitoringv1beta1.Route{
			Receiver:            acm.Spec.Route.Receiver,
			GroupBy:             acm.Spec.Route.GroupBy,
			GroupWait:           acm.Spec.Route.GroupWait,
			GroupInterval:       acm.Spec.Route.GroupInterval,
			RepeatInterval:      acm.Spec.Route.RepeatInterval,
			Matchers:            nil,
			Continue:            acm.Spec.Route.Continue,
			Routes:              acm.Spec.Route.Routes,
			MuteTimeIntervals:   acm.Spec.Route.MuteTimeIntervals,
			ActiveTimeIntervals: acm.Spec.Route.ActiveTimeIntervals,
		}
		for _, m := range acm.Spec.Route.Matchers {
			r.Spec.Route.Matchers = append(r.Spec.Route.Matchers, monitoringv1beta1.Matcher{
				Name:      m.Name,
				Value:     m.Value,
				MatchType: monitoringv1beta1.MatchType(m.MatchType),
			})
		}
	}

	for _, rec := range acm.Spec.Receivers {
		recNew := monitoringv1beta1.Receiver{
			Name: rec.Name,
		}
		for _, w := range rec.WebhookConfigs {
			wc := monitoringv1beta1.WebhookConfig{
				SendResolved: w.SendResolved,
				URL:          w.URL,
				URLSecret:    nil,
				HTTPConfig:   nil,
				MaxAlerts:    w.MaxAlerts,
			}
			if w.URLSecret != nil {
				wc.URLSecret = &monitoringv1beta1.SecretKeySelector{
					Name: w.URLSecret.Name,
					Key:  w.URLSecret.Key,
				}
			}
			if w.HTTPConfig != nil {
				wc.HTTPConfig = &monitoringv1beta1.HTTPConfig{
					Authorization:     w.HTTPConfig.Authorization,
					BasicAuth:         w.HTTPConfig.BasicAuth,
					OAuth2:            w.HTTPConfig.OAuth2,
					BearerTokenSecret: nil,
					TLSConfig:         w.HTTPConfig.TLSConfig,
					ProxyURL:          w.HTTPConfig.ProxyURL,
					FollowRedirects:   w.HTTPConfig.FollowRedirects,
				}
				if w.HTTPConfig.BearerTokenSecret != nil {
					wc.HTTPConfig.BearerTokenSecret = &monitoringv1beta1.SecretKeySelector{
						Name: w.HTTPConfig.BearerTokenSecret.Name,
						Key:  w.HTTPConfig.BearerTokenSecret.Key,
					}
				}
			}
			recNew.WebhookConfigs = append(recNew.WebhookConfigs, wc)
		}
		r.Spec.Receivers = append(r.Spec.Receivers, recNew)
	}

	return r
}

func (r *Reconciler) deletePrometheusRules(ctx context.Context, src *sources.PrometheusRuleSource, toBeDeleted ...*monitoringv1.PrometheusRule) error {
	for _, rule := range toBeDeleted {
		err := r.promClient.MonitoringV1().
			PrometheusRules(rule.Namespace).
			Delete(ctx, rule.Name, metav1.DeleteOptions{
				Preconditions: &metav1.Preconditions{
					UID: &rule.UID,
				},
			})
		if err != nil {
			return fmt.Errorf("failed to delete prometheus rule %s/%s: %w", rule.Namespace, rule.Name, err)
		}

		controller.GetEventRecorder(ctx).
			Eventf(src, corev1.EventTypeNormal, "Deleted", "Deleted Prometheus rule %s/%s", rule.Namespace, rule.Name)
	}
	return nil
}

func (r *Reconciler) createPrometheusRule(ctx context.Context, src *sources.PrometheusRuleSource, rule *monitoringv1.PrometheusRule) error {
	_, err := r.promClient.MonitoringV1().
		PrometheusRules(rule.Namespace).
		Create(ctx, rule, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create prometheus rule %s/%s: %w", rule.Namespace, rule.Name, err)
	}

	controller.GetEventRecorder(ctx).
		Eventf(src, corev1.EventTypeNormal, "Created", "Created Prometheus rule %s/%s", rule.Namespace, rule.Name)

	return nil
}

func (r *Reconciler) updatePrometheusRule(ctx context.Context, src *sources.PrometheusRuleSource, rule *monitoringv1.PrometheusRule) error {
	_, err := r.promClient.MonitoringV1().
		PrometheusRules(rule.Namespace).
		Update(ctx, rule, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update prometheus rule %s/%s: %w", rule.Namespace, rule.Name, err)
	}

	controller.GetEventRecorder(ctx).
		Eventf(src, corev1.EventTypeNormal, "Updated", "Updated Prometheus rule %s/%s", rule.Namespace, rule.Name)

	return nil
}

// HTTPAddress returns the webhook URL
func HTTPAddress(host string, audience *string, object metav1.Object) duckv1.Addressable {
	httpAddress := duckv1.Addressable{
		Name:     pointer.String("http"),
		URL:      apis.HTTP(host),
		Audience: audience,
	}
	httpAddress.URL.Path = fmt.Sprintf("/%s/%s", object.GetNamespace(), object.GetName())
	return httpAddress
}
