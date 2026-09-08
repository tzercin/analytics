// Package tuik provides an authenticated HTTP client for the TÜİK SDMX REST
// API.
//
// The package follows the service-specific authentication, resource paths,
// query parameters, and format names documented at:
//
// https://veriportali.tuik.gov.tr/tr/sdmx-web-service-documentation
//
// It intentionally returns response bodies as streams rather than selecting or
// interpreting a data representation. Callers remain responsible for closing a
// successful response body.
package tuik
