package web

import (
	"net/url"

	"github.com/metno/go-mms/pkg/metaservice"
)

func aboutMMSd() *metaservice.About {
	return &metaservice.About{
		Name:           "MMSd REST API",
		Description:    "Receive, list and publish events coming from a production hub.",
		Responsible:    "Production hub team",
		Documentation:  &url.URL{Path: "/"},
		TermsOfService: &url.URL{Path: "/docs/termsofservice"},
	}
}
