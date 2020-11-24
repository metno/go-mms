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
	"fmt"
	"net/http"
	"net/url"
)

// About contains "static" metadata information about a service.
// Use it to display healthz, discovery and internalstatus etc.
type About struct {
	Name           string
	Description    string
	Responsible    string
	TermsOfService *url.URL
	Documentation  *url.URL
}

type WebAPISchemaOrg struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	ToS           string   `json:"termsOfService"`
	Documentation string   `json:"documentation"`
	Provider      Provider `json:"provider"`
}

type Provider struct {
	Ldtype string `json:"@type"`
	Name   string `json:"name"`
}

func aboutMMSd() *About {
	return &About{
		Name:           "MMSd REST API",
		Description:    "Receive, list and publish events coming from a production hub.",
		Responsible:    "Production hub team",
		Documentation:  &url.URL{Path: "/"},
		TermsOfService: &url.URL{Path: "/docs/termsofservice"},
	}
}

func AboutHandler(about *About) func(httpRespW http.ResponseWriter, httpReq *http.Request) {
	return func(httpRespW http.ResponseWriter, httpReq *http.Request) {
		baseURL := fmt.Sprintf("%s://%s", httpReq.URL.Scheme, httpReq.Host)
		response, err := EncodeAboutAsSchemaOrg(about, baseURL)
		if err != nil {
			http.Error(httpRespW, fmt.Sprintf("Failed to encode %s. Something very wrong is going on.", httpReq.URL.Path),
				http.StatusServiceUnavailable)
			return
		}
		httpRespW.Header().Set("Link", "<https://schema.org/WebAPI.jsonld>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"")
		httpRespW.Header().Set("Cache-Control", "max-age=60")
		httpRespW.Header().Set("Content-Type", "application/json")

		httpRespW.Write(response)
	}
}

func EncodeAboutAsSchemaOrg(about *About, baseURL string) ([]byte, error) {
	payload, err := json.Marshal(
		WebAPISchemaOrg{
			Name:          about.Name,
			Description:   about.Description,
			ToS:           fmt.Sprintf("%s%s", baseURL, about.TermsOfService),
			Documentation: fmt.Sprintf("%s%s", baseURL, about.Documentation),
			Provider: Provider{
				Ldtype: "Organization",
				Name:   "Met Norway",
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return payload, nil
}

func EncodeDiscoveryAsISO19115(about *About) string {
	return ""
}
