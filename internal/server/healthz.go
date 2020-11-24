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
	return func(httpRespW http.ResponseWriter, httpReq *http.Request) {
		healthz, err := check()
		if err != nil {
			http.Error(httpRespW, "Could not check the health of the service.", http.StatusInternalServerError)
		}
		healthz.respond(httpRespW, httpReq)
	}
}

func (hltz *Healthz) respond(httpRespW http.ResponseWriter, httpReq *http.Request) {
	response, err := hltz.encodeJSONLD()
	if err != nil {
		http.Error(httpRespW, "Failed to encode health. Something very wrong is going on.",
			http.StatusInternalServerError)
		return
	}
	httpRespW.Header().Set("Link", "Link: <https://schema.met.no/contexts/healthz.jsonld>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"")
	httpRespW.Header().Set("Cache-Control", "max-age=360")
	httpRespW.Header().Set("Content-Type", "application/json")
	if hltz.Status != HealthzStatusHealthy {
		httpRespW.WriteHeader(http.StatusServiceUnavailable)
	}
	httpRespW.Write(response)
}

func (healthz *Healthz) encodeJSONLD() ([]byte, error) {
	ld := &healthzJSONLD{
		Status:      healthz.Status.String(),
		Description: healthz.Description,
	}

	payload, err := json.Marshal(ld)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func (hltzStatus HealthzStatus) String() string {
	switch hltzStatus {
	case HealthzStatusHealthy:
		return "healthy"
	case HealthzStatusUnhealthy:
		return "unhealthy"
	case HealthzStatusCritical:
		return "critical"
	}

	return "unknown"
}
