/*
Copyright 2019 The Knative Authors.

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

package profiling

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/metrics"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

func TestUpdateFromConfigMap(t *testing.T) {
	observabilityConfigTests := []struct {
		name                   string
		wantEnabledAtStartup   bool
		wantEnabledAfterUpdate bool
		wantStatusCode         int
		initialConfig          *corev1.ConfigMap
		updatedConfig          *corev1.ConfigMap
	}{{
		name:                   "observability with profiling disabled",
		wantEnabledAtStartup:   false,
		wantEnabledAfterUpdate: false,
		wantStatusCode:         http.StatusNotFound,
		initialConfig: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{},
		},
		updatedConfig: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{
				"profiling.enable": "false",
			},
		},
	}, {
		name:                   "observability config with profiling enabled",
		wantEnabledAtStartup:   false,
		wantEnabledAfterUpdate: true,
		wantStatusCode:         http.StatusOK,
		initialConfig: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{},
		},
		updatedConfig: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{
				"profiling.enable": "true",
			},
		},
	}, {
		name:                   "observability config with unparseable value",
		wantEnabledAtStartup:   false,
		wantEnabledAfterUpdate: false,
		wantStatusCode:         http.StatusNotFound,
		initialConfig: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{},
		},
		updatedConfig: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{
				"profiling.enable": "get me some profiles",
			},
		},
	}, {
		name:                   "observability config with profiling enabled at startup",
		wantEnabledAtStartup:   true,
		wantEnabledAfterUpdate: false,
		wantStatusCode:         http.StatusNotFound,
		initialConfig: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{
				"profiling.enable": "true",
			},
		},
		updatedConfig: &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: system.Namespace(),
				Name:      metrics.ConfigMapName(),
			},
			Data: map[string]string{},
		},
	}}

	for _, tt := range observabilityConfigTests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(zap.NewNop().Sugar(), tt.initialConfig)

			if handler.enabled != tt.wantEnabledAtStartup {
				t.Fatalf("Test: %q; want %v, but got %v", tt.name, tt.wantEnabledAtStartup, handler.enabled)
			}

			handler.UpdateFromConfigMap(tt.updatedConfig)

			if handler.enabled != tt.wantEnabledAfterUpdate {
				t.Fatalf("Test: %q; want %v, but got %v", tt.name, tt.wantEnabledAfterUpdate, handler.enabled)
			}

			req, err := http.NewRequest(http.MethodGet, "/debug/pprof/", nil)
			if err != nil {
				t.Fatal("Error creating request:", err)
			}

			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatusCode {
				t.Errorf("StatusCode: %v, want: %v", rr.Code, tt.wantStatusCode)
			}
		})
	}
}
