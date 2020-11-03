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

package metaservice

import (
	"encoding/json"
	"net/http"
)

type Healthz struct {
	Status      HealthzStatus
	Description string
}

type HealthzStatus int

const (
	HealthzStatusHealthy   HealthzStatus = 0
	HealthzStatusUnhealthy               = 1
	HealthzStatusCritical                = 2
)

type healthzJSONLD struct {
	Status      string `json:"status"`
	Description string `json:"description"`
}

// HealthzHandler runs the callback function check and serializes and sends the result of that check.
func HealthzHandler(check func() (*Healthz, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		healthz, err := check()
		if err != nil {
			http.Error(w, "Could not check the health of the service.", http.StatusInternalServerError)
		}
		healthz.respond(w, r)
	}
}

func (h *Healthz) respond(w http.ResponseWriter, r *http.Request) {
	response, err := h.encodeJSONLD()
	if err != nil {
		http.Error(w, "Failed to encode health. Something very wrong is going on.",
			http.StatusInternalServerError)
		return
	}
	w.Header().Set("Link", "Link: <https://schema.met.no/contexts/healthz.jsonld>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"")
	w.Header().Set("Cache-Control", "max-age=360")
	w.Header().Set("Content-Type", "application/json")
	if h.Status != HealthzStatusHealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	w.Write(response)
}

func (h *Healthz) encodeJSONLD() ([]byte, error) {
	ld := &healthzJSONLD{
		Status:      h.Status.String(),
		Description: h.Description,
	}

	payload, err := json.Marshal(ld)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (hs HealthzStatus) String() string {
	switch hs {
	case HealthzStatusHealthy:
		return "healthy"
	case HealthzStatusUnhealthy:
		return "unhealthy"
	case HealthzStatusCritical:
		return "critical"
	}

	return "unknown"
}
