/*
  Copyright 2020 MET Norway

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

package server

import (
	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type MetricsOpts struct {
	Name            string
	Description     string
	ResponseBuckets []float64
}

type metrics struct {
	registry *prometheus.Registry
	counter  *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func NewServiceMetrics(opts MetricsOpts) *metrics {
	registry := prometheus.NewRegistry()

	counter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: fmt.Sprintf("%s_requests_total", opts.Name),
			Help: opts.Description,
		},
		[]string{"endpoint", "code", "method"},
	)

	// duration is partitioned by the HTTP method and handler. It uses custom
	// buckets based on the expected request duration.
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    fmt.Sprintf("%s_request_duration_seconds", opts.Name),
			Help:    "A histogram of latencies for requests.",
			Buckets: opts.ResponseBuckets,
		},
		[]string{"endpoint", "method"},
	)
	registry.MustRegister(counter)
	registry.MustRegister(duration)

	return &metrics{
		registry: registry,
		counter:  counter,
		duration: duration,
	}
}

func (m *metrics) Endpoint(endpoint string, h http.HandlerFunc) http.HandlerFunc {
	return promhttp.InstrumentHandlerDuration(m.duration.MustCurryWith(prometheus.Labels{"endpoint": endpoint}),
		promhttp.InstrumentHandlerCounter(m.counter.MustCurryWith(prometheus.Labels{"endpoint": endpoint}), h),
	)
}

func (m *metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
