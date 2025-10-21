/*
Copyright 2020–2021 MET Norway

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
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	gorilla "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/nats-io/nats.go"
	"github.com/rakyll/statik/fs"

	"github.com/metno/go-mms/pkg/mms"
	_ "github.com/metno/go-mms/pkg/statik"
)

// Version and build information
type Version struct {
	Version string
	Commit  string
	Date    string
}

// Service is a struct that wires up all data that is needed for this service to run.
type Service struct {
	eventsDB        *sql.DB
	stateDB         *sql.DB
	about           *About
	htmlTemplates   *template.Template
	Router          *mux.Router
	NatsURL         string
	NatsCredentials nats.Option
	NatsLocal       bool
	Metrics         *metrics
	Productstatus   *Productstatus
	Version         Version
}

// HTTPServerError is used when the server fails to return a correct response to the user.
type HTTPServerError struct {
	ErrMsg string `json:"error"`
}

// NewService creates a service struct, containing all that is needed for a mmsd server to run.
func NewService(templates *template.Template, eventsDB *sql.DB, stateDB *sql.DB, natsURL string, natsCredentials nats.Option, version Version, natsLocal bool) *Service {
	m := NewServiceMetrics(MetricsOpts{})

	service := Service{
		eventsDB:        eventsDB,
		stateDB:         stateDB,
		about:           aboutMMSd(version),
		htmlTemplates:   templates,
		Router:          mux.NewRouter(),
		NatsURL:         natsURL,
		NatsCredentials: natsCredentials,
		NatsLocal:       natsLocal,
		Metrics:         m,
		Productstatus:   NewProductstatus(m),
		Version:         version,
	}
	service.setRoutes()

	return &service
}

func (service *Service) setRoutes() {

	statikFS, err := fs.New()
	if err != nil {
		log.Fatal(err)
	}
	//var statikFS embed.FS
	// Events
	service.Router.HandleFunc("/api/v1/events", service.Metrics.Endpoint("/v1/events", service.eventsHandler)).Methods("GET")
	service.Router.Handle("/api/v1/events", proxyHeaders(service.postEventHandler)).Methods("POST")

	// Health of the service
	service.Router.HandleFunc("/api/v1/healthz", HealthzHandler(service.checkHealthz))

	// Service discovery metadata for the world
	service.Router.Handle("/api/v1/about", proxyHeaders(AboutHandler(service.about)))

	// Metrics of the service(service) for this app.
	service.Router.Handle("/metrics", service.Metrics.Handler())

	// Documentation of the service(service)
	service.Router.HandleFunc("/docs/{page}", service.docsHandler)

	//http.HandleFunc("/", mockProductEvent)
	service.Router.HandleFunc("/mockevent", MockProductEvent)

	// Static assets.
	service.Router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(statikFS)))

	// Send root path of the http service to the docs index page.
	service.Router.HandleFunc("/", service.docsHandler)
}

// proxyHeaders is a http handler function for setting scheme and host correctly when behind a proxy.
// Usually needed when the response consists of urls to the service.
func proxyHeaders(next func(httpRespW http.ResponseWriter, httpReq *http.Request)) http.Handler {
	setSchemeIfEmpty := func(httpRespW http.ResponseWriter, httpReq *http.Request) {
		if httpReq.URL.Scheme == "" {
			httpReq.URL.Scheme = "http"
		}
		next(httpRespW, httpReq)
	}
	return gorilla.ProxyHeaders(http.HandlerFunc(setSchemeIfEmpty))
}

const eventsApiResponseTimeoutSecs = 15

func (service *Service) eventsHandler(httpRespW http.ResponseWriter, httpReq *http.Request) {
	dbCtx, cancel := context.WithTimeout(httpReq.Context(), time.Duration(eventsApiResponseTimeoutSecs)*time.Second)
	defer cancel()

	events, err := service.GetAllEvents(dbCtx)
	if err != nil {
		serverErrorResponse(err, httpRespW, httpReq)
		return
	}

	payload, err := json.Marshal(events)
	if err != nil {
		http.Error(httpRespW, "Failed to serialize data.", http.StatusInternalServerError)
		return
	}
	okResponse(payload, httpRespW, httpReq)
}

// html docs generated from templates.
func (service *Service) docsHandler(httpRespW http.ResponseWriter, httpReq *http.Request) {
	params := mux.Vars(httpReq)
	page, exists := params["page"]

	var err error
	if !exists {
		err = service.htmlTemplates.ExecuteTemplate(httpRespW, "index", service.about)
	} else {
		err = service.htmlTemplates.ExecuteTemplate(httpRespW, page, service.about)
	}
	if err != nil {
		http.Error(httpRespW, err.Error(), http.StatusInternalServerError)
	}
}

// Post an event to the API
func (service *Service) postEventHandler(httpRespW http.ResponseWriter, httpReq *http.Request) {

	if httpReq.Method != "POST" {
		httpRespW.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	log.Print("Post started")
	var err error
	var validKey bool
	var natsUser string
	var pEvent mms.ProductEvent
	var payLoad []byte
	var postCredentials nats.Option
	apiKey := httpReq.Header.Get("Api-Key")
	if apiKey == "" {
		http.Error(httpRespW, "API key invalid or missing", http.StatusUnauthorized)
		log.Print("unauthorized: API key invalid or missing")
		return
	}
	if service.NatsLocal {
		validKey, err = ValidateApiKey(service.stateDB, apiKey)
		postCredentials = service.NatsCredentials
	} else {
		validKey, natsUser, err = ValidateJWTKey(service.stateDB, apiKey)
		postCredentials = nats.UserCredentials(natsUser)
	}
	if err != nil {
		log.Printf("Failed to validate key: %s", err)
	}
	queueName := httpReq.Header.Get("Queue-Name")
	if queueName == "" {
		queueName = "mms"
	}

	if !validKey {
		http.Error(httpRespW, "Unauthorized API key submitted", http.StatusUnauthorized)
		log.Print("unauthorized: API key not accepted")
		return
	}

	payLoad, err = ioutil.ReadAll(httpReq.Body)
	if err != nil {
		http.Error(httpRespW, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		log.Printf("failed reading request body: %v", err)
		return
	}

	err = json.Unmarshal(payLoad, &pEvent)
	if err != nil {
		http.Error(httpRespW, fmt.Sprintf("%v", err), http.StatusBadRequest)
		log.Printf("failed to unmarshal request body: %v", err)
		return
	}

	if pEvent.ProductionHub == "" {
		http.Error(httpRespW, "ProductionHub must be given", http.StatusBadRequest)
		log.Print("ProductionHub must be given")
		return
	}

	err = saveProductEvent(service.eventsDB, &pEvent)
	if err != nil {
		log.Printf("could not save to database: %v", err)
		http.Error(httpRespW, fmt.Sprintf("%v", err), http.StatusInternalServerError)
		return
	}

	err = mms.MakeProductEvent(service.NatsURL, postCredentials, &pEvent, queueName, service.NatsLocal)
	if err != nil {
		http.Error(httpRespW, fmt.Sprintf("%v", err), http.StatusBadRequest)
		log.Printf("failed to create ProductEvent: %v", err)
		return
	}

	httpRespW.WriteHeader(http.StatusCreated)
	service.Productstatus.PushEvent(pEvent)
	log.Print("Post ended")

}

// checkHealthz is supplied to HealthzHandler as a callback function.
func (service *Service) checkHealthz() (*Healthz, error) {
	return &Healthz{
		Status:      HealthzStatusHealthy,
		Description: "No deps, so everything is ok all the time.",
	}, nil
}

func okResponse(payload []byte, httpRespW http.ResponseWriter, httpReq *http.Request) {
	httpRespW.Header().Set("Cache-Control", "max-age=10")
	httpRespW.Header().Set("Content-Type", "application/json")
	_, err := httpRespW.Write(payload)
	if err != nil {
		log.Printf("failed to send response to req %q: %s", httpReq.URL, err)
	}
}

func serverErrorResponse(errMsg error, httpRespW http.ResponseWriter, httpReq *http.Request) {
	errResponse := HTTPServerError{
		ErrMsg: errMsg.Error(),
	}

	payload, err := json.Marshal(errResponse)
	if err != nil {
		http.Error(httpRespW, "failed to serialize data", http.StatusInternalServerError)
		return
	}
	httpRespW.WriteHeader(http.StatusServiceUnavailable)

	httpRespW.Header().Set("Content-Type", "application/json")
	_, err = httpRespW.Write(payload)
	if err != nil {
		log.Printf("failed to send response to req %q: %s", httpReq.URL, err)
	}
}
