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

// Package server provides a http service struct for a events service.
// All request and response structs and handlers for this service are located in this package.
package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"time"

	gorilla "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/rakyll/statik/fs"

	"github.com/metno/go-mms/pkg/metaservice"
	"github.com/metno/go-mms/pkg/middleware"
	_ "github.com/metno/go-mms/pkg/statik"
)

// Service is a struct that wires up all data that is needed for this service to run.
type Service struct {
	cacheDB        *sql.DB
	about          *metaservice.About
	htmlTemplates  *template.Template
	staticFilesDir string
	Router         *mux.Router
}

// HTTPServerError is used when the server fails to return a correct response to the user.
type HTTPServerError struct {
	ErrMsg string `json:"error"`
}

// NewService creates a service struct, containing all that is needed for a mmsd server to run.
func NewService(templates *template.Template, staticFilesDir string, cacheDB *sql.DB) *Service {
	service := Service{
		cacheDB:        cacheDB,
		about:          aboutMMSd(),
		htmlTemplates:  templates,
		staticFilesDir: staticFilesDir,
		Router:         mux.NewRouter(),
	}
	service.routes()

	return &service
}

func (service *Service) routes() {
	var metrics = middleware.NewServiceMetrics(middleware.MetricsOpts{
		Name:            "events",
		Description:     "MMSd production hub events.",
		ResponseBuckets: []float64{0.001, 0.002, 0.1, 0.5},
	})

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}

	// The Eventscache endpoint.
	service.Router.HandleFunc("/api/v1/events", metrics.Endpoint("/v1/events", service.eventsHandler))

	// Health of the service
	service.Router.HandleFunc("/api/v1/healthz", metaservice.HealthzHandler(service.checkHealthz))

	// Service discovery metadata for the world
	service.Router.Handle("/api/v1/about", proxyHeaders(metaservice.AboutHandler(service.about)))

	// Metrics of the service(service) for this app.
	service.Router.Handle("/metrics", metrics.Handler())

	// Documentation of the service(service)
	service.Router.HandleFunc("/docs/{page}", service.docsHandler)

	//http.HandleFunc("/", mockProductEvent)
	service.Router.HandleFunc("/mockevent", metaservice.MockProductEvent)

	// Swagger UI
	swui := http.StripPrefix("/swaggerui", http.FileServer(http.Dir("./static/swaggerui/")))
	service.Router.PathPrefix("/swaggerui").Handler(swui)

	// Static assets.
	service.Router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(statikFS)))

	// Send root path of the http service to the docs index page.
	service.Router.HandleFunc("/", service.docsHandler)
}

// proxyHeaders is a http handler middleware function for setting scheme and host correctly when behind a proxy.
// Usually needed when the response consists of urls to the service.
func proxyHeaders(next func(httpResp http.ResponseWriter, httpReq *http.Request)) http.Handler {
	setSchemeIfEmpty := func(httpResp http.ResponseWriter, httpReq *http.Request) {
		if httpReq.URL.Scheme == "" {
			httpReq.URL.Scheme = "http"
		}
		next(httpResp, httpReq)
	}
	return gorilla.ProxyHeaders(http.HandlerFunc(setSchemeIfEmpty))
}

func (service *Service) eventsHandler(httpResp http.ResponseWriter, httpReq *http.Request) {
	dbCtx, cancel := context.WithTimeout(httpReq.Context(), 5*time.Second)
	defer cancel()

	events, err := service.GetAllEvents(dbCtx)
	if err != nil {
		serverErrorResponse(err, httpResp, httpReq)
		return
	}

	payload, err := json.Marshal(events)
	if err != nil {
		http.Error(httpResp, "Failed to serialize data.", http.StatusInternalServerError)
		return
	}
	okResponse(payload, httpResp, httpReq)
}

// html docs generated from templates.
func (service *Service) docsHandler(httpResp http.ResponseWriter, httpReq *http.Request) {
	params := mux.Vars(httpReq)
	page, exists := params["page"]

	var err error
	if exists != true {
		err = service.htmlTemplates.ExecuteTemplate(httpResp, "index", service.about)
	} else {
		err = service.htmlTemplates.ExecuteTemplate(httpResp, page, service.about)
	}
	if err != nil {
		http.Error(httpResp, err.Error(), http.StatusInternalServerError)
	}
}

// checkHealthz is supplied to metaservice.HealthzHandler as a callback function.
func (service *Service) checkHealthz() (*metaservice.Healthz, error) {
	return &metaservice.Healthz{
		Status:      metaservice.HealthzStatusHealthy,
		Description: "No deps, so everything is ok all the time.",
	}, nil
}

func okResponse(payload []byte, httpResp http.ResponseWriter, httpReq *http.Request) {
	httpResp.Header().Set("Cache-Control", "max-age=10")
	httpResp.Header().Set("Content-Type", "application/json")
	_, err := httpResp.Write(payload)
	if err != nil {
		log.Printf("could send response to req %q: %s", httpReq.URL, err)
	}
}

func serverErrorResponse(errMsg error, httpResp http.ResponseWriter, httpReq *http.Request) {
	errResponse := HTTPServerError{
		ErrMsg: errMsg.Error(),
	}

	payload, err := json.Marshal(errResponse)
	if err != nil {
		http.Error(httpResp, "Failed to serialize data.", http.StatusInternalServerError)
		return
	}
	httpResp.WriteHeader(http.StatusServiceUnavailable)

	httpResp.Header().Set("Content-Type", "application/json")
	_, err = httpResp.Write(payload)
	if err != nil {
		log.Printf("could send response to req %q: %s", httpReq.URL, err)
	}
}
