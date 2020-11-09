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

func (s *Service) routes() {
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
	s.Router.HandleFunc("/api/v1/events", metrics.Endpoint("/v1/events", s.eventsHandler))

	// Health of the service
	s.Router.HandleFunc("/api/v1/healthz", metaservice.HealthzHandler(s.checkHealthz))

	// Service discovery metadata for the world
	s.Router.Handle("/api/v1/about", proxyHeaders(metaservice.AboutHandler(s.about)))

	// Metrics of the service(s) for this app.
	s.Router.Handle("/metrics", metrics.Handler())

	// Documentation of the service(s)
	s.Router.HandleFunc("/docs/{page}", s.docsHandler)

	//http.HandleFunc("/", mockProductEvent)
	s.Router.HandleFunc("/mockevent", metaservice.MockProductEvent)

	// Swagger UI
	swui := http.StripPrefix("/swaggerui", http.FileServer(http.Dir("./static/swaggerui/")))
	s.Router.PathPrefix("/swaggerui").Handler(swui)

	// Static assets.
	s.Router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(statikFS)))

	// Send root path of the http service to the docs index page.
	s.Router.HandleFunc("/", s.docsHandler)
}

// proxyHeaders is a http handler middleware function for setting scheme and host correctly when behind a proxy.
// Usually needed when the response consists of urls to the service.
func proxyHeaders(next func(w http.ResponseWriter, r *http.Request)) http.Handler {
	setSchemeIfEmpty := func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Scheme == "" {
			r.URL.Scheme = "http"
		}
		next(w, r)
	}
	return gorilla.ProxyHeaders(http.HandlerFunc(setSchemeIfEmpty))
}

func (s *Service) eventsHandler(w http.ResponseWriter, r *http.Request) {
	dbCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	events, err := s.GetAllEvents(dbCtx)
	if err != nil {
		serverErrorResponse(err, w, r)
		return
	}

	payload, err := json.Marshal(events)
	if err != nil {
		http.Error(w, "Failed to serialize data.", http.StatusInternalServerError)
		return
	}
	okResponse(payload, w, r)
}

// html docs generated from templates.
func (s *Service) docsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	page, exists := params["page"]

	var err error
	if exists != true {
		err = s.htmlTemplates.ExecuteTemplate(w, "index", s.about)
	} else {
		err = s.htmlTemplates.ExecuteTemplate(w, page, s.about)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// checkHealthz is supplied to metaservice.HealthzHandler as a callback function.
func (s *Service) checkHealthz() (*metaservice.Healthz, error) {
	return &metaservice.Healthz{
		Status:      metaservice.HealthzStatusHealthy,
		Description: "No deps, so everything is ok all the time.",
	}, nil
}

func okResponse(payload []byte, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "max-age=10")
	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(payload)
	if err != nil {
		log.Printf("could send response to req %q: %s", r.URL, err)
	}
}

func serverErrorResponse(errMsg error, w http.ResponseWriter, r *http.Request) {
	errResponse := HTTPServerError{
		ErrMsg: errMsg.Error(),
	}

	payload, err := json.Marshal(errResponse)
	if err != nil {
		http.Error(w, "Failed to serialize data.", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusServiceUnavailable)

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(payload)
	if err != nil {
		log.Printf("could send response to req %q: %s", r.URL, err)
	}
}
