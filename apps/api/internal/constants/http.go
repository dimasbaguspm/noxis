package constants

import "net/http"

// Header names: spelled once here, so a client, a middleware and a driver's HTTP call agree on one spelling.
const (
	HeaderAuthorization = "Authorization"
	HeaderContentType   = "Content-Type"

	HeaderAccessControlAllowOrigin  = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders = "Access-Control-Allow-Headers"

	// HeaderSchedulerTimestamp and HeaderSchedulerSignature sign the external scheduler trigger.
	HeaderSchedulerTimestamp = "X-Noxis-Timestamp"
	HeaderSchedulerSignature = "X-Noxis-Signature"
)

// Authorization schemes: the word a credential is prefixed with after the header name.
const (
	SchemeBearer = "Bearer"
	SchemeSplunk = "Splunk"
)

// Media types written by the API and by the drivers' own HTTP clients.
const (
	MediaTypeJSON = "application/json"
	MediaTypeHTML = "text/html; charset=utf-8"
	MediaTypeForm = "application/x-www-form-urlencoded"
)

// CORS methods is the preflight allow-list: the verbs the console actually uses.
var CORSMethods = []string{http.MethodGet, http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodOptions}

// CORSHeaders is the preflight header allow-list: the two headers a console request carries.
var CORSHeaders = []string{HeaderContentType, HeaderAuthorization}
