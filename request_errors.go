package main

import (
	"net/http"
)

type notFoundHandler struct {
}

func (h notFoundHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	e := newException("ResourceNotFoundException", "Resource not found")
	e.write(w)

}

type methodNotAllowedHandler struct{}

func (h methodNotAllowedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	e := newException("MethodNotAllowedException", "Method not allowed")
	e.write(w)

}
