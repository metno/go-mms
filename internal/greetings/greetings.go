// Package greetings provides a http service struct for a greeting service.
// All request and response structs and handlers for this service are located in this package.
package greetings

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"

	gorilla "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/metno/go-mms/pkg/metaservice"
	"github.com/metno/go-mms/pkg/middleware"
)

type service struct {
	about          *metaservice.About
	htmlTemplates  *template.Template
	staticFilesDir string
	InternalRouter *mux.Router
	ExternalRouter *mux.Router
}

type Greeting struct {
	Greeting string `json:"greeting"`
}

type GreetingBeta struct {
	Greeting string   `json:"greeting"`
	Geometry geometry `json:"geometry"`
}

type geometry struct {
	Type        string
	Coordinates []float32 `json:"coordinates"`
}

type ServerError struct {
	ErrMsg string `json:"error"`
}

func NewService(templates *template.Template, staticFilesDir string) *service {
	service := service{
		about:          aboutHello(),
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
		Name:            "hello",
		Description:     "Hello REST service at MET Norway.",
		ResponseBuckets: []float64{0.001, 0.002, 0.1, 0.5},
	})

	// Beta version of hello service endpoint.
	s.ExternalRouter.HandleFunc("/api/v1beta/hello/{who}", metrics.Endpoint("/v1beta/hello", s.helloBetaHandler))

	// The hello service endpoint.
	s.ExternalRouter.HandleFunc("/api/v1/hello/{who}", metrics.Endpoint("/v1/hello", s.helloHandler))

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

func (s *service) helloHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	greeting := &Greeting{
		Greeting: fmt.Sprintf("Hello %s.", params["who"]),
	}
	var err error
	if err != nil {
		serverErrorResponse(err, w, r)
		return
	}

	payload, err := json.Marshal(greeting)
	if err != nil {
		http.Error(w, "Failed to serialize data.", http.StatusInternalServerError)
		return
	}
	okResponse(payload, w, r)
}

func (s *service) helloBetaHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	greeting := &GreetingBeta{
		Greeting: fmt.Sprintf("Hello %s.", params["who"]),
		Geometry: geometry{
			Type:        "POINT",
			Coordinates: []float32{60, 11, 112},
		},
	}
	var err error
	if err != nil {
		serverErrorResponse(err, w, r)
		return
	}

	payload, err := json.Marshal(greeting)
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
	w.Header().Set("Cache-Control", "max-age=86400")
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
