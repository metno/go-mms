// Package api provides a http service struct for a events service.
// All request and response structs and handlers for this service are located in this package.
package api

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"

	gorilla "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/metno/go-mms/pkg/metaservice"
	"github.com/metno/go-mms/pkg/middleware"
	"github.com/metno/go-mms/pkg/mms"
)

type service struct {
	about          *metaservice.About
	htmlTemplates  *template.Template
	staticFilesDir string
	InternalRouter *mux.Router
	ExternalRouter *mux.Router
}

type ServerError struct {
	ErrMsg string `json:"error"`
}

func NewService(templates *template.Template, staticFilesDir string) *service {
	service := service{
		about:          aboutMMSd(),
		htmlTemplates:  templates,
		staticFilesDir: staticFilesDir,
		InternalRouter: mux.NewRouter(),
		ExternalRouter: mux.NewRouter(),
	}
	service.routes()

	return &service
}

func (s *service) routes() {
	var metrics = middleware.NewServiceMetrics(middleware.MetricsOpts{
		Name:            "events",
		Description:     "MMSd production hub events.",
		ResponseBuckets: []float64{0.001, 0.002, 0.1, 0.5},
	})

	// The Eventscache endpoint.
	s.ExternalRouter.HandleFunc("/api/v1/events", metrics.Endpoint("/v1/events", s.eventsHandler))

	// Health of the service
	s.ExternalRouter.HandleFunc("/api/v1/healthz", metaservice.HealthzHandler(s.checkHealthz))

	// Service discovery metadata for the world
	s.ExternalRouter.Handle("/api/v1/about", proxyHeaders(metaservice.AboutHandler(s.about)))

	// Metrics of the service(s) for this app.
	s.InternalRouter.Handle("/metrics", metrics.Handler())

	// Documentation of the service(s)
	s.ExternalRouter.HandleFunc("/docs/{page}", s.docsHandler)

	//http.HandleFunc("/", mockProductEvent)
	s.ExternalRouter.HandleFunc("/mockevent", metaservice.MockProductEvent)

	// Swagger UI
	swui := http.StripPrefix("/swaggerui", http.FileServer(http.Dir("./static/swaggerui/")))
	s.ExternalRouter.PathPrefix("/swaggerui").Handler(swui)

	// Static assets.
	s.ExternalRouter.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(s.staticFilesDir))))

	// Send root path of the http service to the docs index page.
	s.ExternalRouter.HandleFunc("/", s.docsHandler)
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

func (s *service) eventsHandler(w http.ResponseWriter, r *http.Request) {
	events := []mms.ProductEvent{
		{},
	}
	var err error
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
func (s *service) docsHandler(w http.ResponseWriter, r *http.Request) {
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
func (s *service) checkHealthz() (*metaservice.Healthz, error) {
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
	errResponse := ServerError{
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
