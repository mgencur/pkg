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

package profiling

import (
	"net/http"
	"net/http/pprof"
	"strconv"
	"sync"

	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
)

const (
	// profilingPort is the port where we expose profiling information if profiling is enabled
	profilingPort = ":8008"

	// profilingKey is the name of the key in config-observability config map that indicates whether profiling
	// is enabled of disabled
	profilingKey = "profiling.enable"
)

// Handler holds the main HTTP handler and a flag indicating
// whether the handler is active
type Handler struct {
	enabled bool
	handler http.Handler
	log     *zap.SugaredLogger
	mutex   sync.Mutex
}

// NewHandler create a new ProfilingHandler which serves runtime profiling data
// according to the given context path
func NewHandler(logger *zap.SugaredLogger) *Handler {
	const pprofPrefix = "/debug/pprof/"

	mux := http.NewServeMux()
	mux.HandleFunc(pprofPrefix, pprof.Index)
	mux.HandleFunc(pprofPrefix+"cmdline", pprof.Cmdline)
	mux.HandleFunc(pprofPrefix+"profile", pprof.Profile)
	mux.HandleFunc(pprofPrefix+"symbol", pprof.Symbol)
	mux.HandleFunc(pprofPrefix+"trace", pprof.Trace)

	return &Handler{
		enabled: false,
		handler: mux,
		log:     logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if h.enabled {
		h.handler.ServeHTTP(w, r)
	} else {
		http.NotFoundHandler().ServeHTTP(w, r)
	}
}

// UpdateFromConfigMap modifies the Enabled flag in the Handler
// according to the value in the given ConfigMap
func (h *Handler) UpdateFromConfigMap(configMap *corev1.ConfigMap) {
	profiling, ok := configMap.Data[profilingKey]
	if !ok {
		return
	}
	enabled, err := strconv.ParseBool(profiling)
	if err != nil {
		h.log.Errorw("Failed to update profiling", zap.Error(err))
		return
	}
	h.log.Infof("Profiling enabled: %t", enabled)

	h.mutex.Lock()
	defer h.mutex.Unlock()
	h.enabled = enabled
}

// NewServer creates a new http server that exposes profiling data using the
// HTTP handler that is passed as an argument
func NewServer(handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    profilingPort,
		Handler: handler,
	}
}
