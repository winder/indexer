// Package common provides primitives to interact the openapi HTTP API.
//
// Code generated by github.com/algorand/oapi-codegen DO NOT EDIT.
package common

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
	"net/http"
	"strings"
)

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Returns 200 if healthy.
	// (GET /health)
	MakeHealthCheck(ctx echo.Context) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// MakeHealthCheck converts echo context to params.
func (w *ServerInterfaceWrapper) MakeHealthCheck(ctx echo.Context) error {

	validQueryParams := map[string]bool{
		"pretty": true,
	}

	// Check for unknown query parameters.
	for name, _ := range ctx.QueryParams() {
		if _, ok := validQueryParams[name]; !ok {
			return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Unknown parameter detected: %s", name))
		}
	}

	var err error

	// Invoke the callback with all the unmarshalled arguments
	err = w.Handler.MakeHealthCheck(ctx)
	return err
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}, si ServerInterface, m ...echo.MiddlewareFunc) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.GET("/health", wrapper.MakeHealthCheck, m...)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file.
func GetSwagger() (*openapi3.Swagger, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}

	swagger, err := openapi3.NewSwaggerLoader().LoadSwaggerFromData(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("error loading Swagger: %s", err)
	}
	return swagger, nil
}
