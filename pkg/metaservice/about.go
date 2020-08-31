package metaservice

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

func AboutHandler(about *About) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		baseURL := fmt.Sprintf("%s://%s", r.URL.Scheme, r.Host)
		response, err := EncodeAboutAsSchemaOrg(about, baseURL)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to encode %s. Something very wrong is going on.", r.URL.Path),
				http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Link", "<https://schema.org/WebAPI.jsonld>; rel=\"http://www.w3.org/ns/json-ld#context\"; type=\"application/ld+json\"")
		w.Header().Set("Cache-Control", "max-age=60")
		w.Header().Set("Content-Type", "application/json")

		w.Write(response)
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
