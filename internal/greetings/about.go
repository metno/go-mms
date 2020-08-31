package greetings

import (
	"net/url"

	"github.com/metno/go-mms/pkg/metaservice"
)

func aboutHello() *metaservice.About {
	return &metaservice.About{
		Name:           "Hello REST service",
		Description:    "The purpose of this service is to deliver a nice greeting.",
		Responsible:    "Hello Product Team <hello@met.no>",
		Documentation:  &url.URL{Path: "/"},
		TermsOfService: &url.URL{Path: "/docs/termsofservice"},
	}
}
