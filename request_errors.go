package main

import (
    "net/http"
)

type NotFoundHandler struct {
}
func (h NotFoundHandler) ServeHTTP(w http.ResponseWriter,r *http.Request) {

	e := NewException("ResourceNotFoundException", "Resource not found")
	e.Write(w)

}

type MethodNotAllowedHandler struct {}

func (h MethodNotAllowedHandler) ServeHTTP(w http.ResponseWriter,r *http.Request) {

	e := NewException("MethodNotAllowedException", "Method not allowed")
	e.Write(w)

}
