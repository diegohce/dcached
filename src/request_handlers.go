package main

import (
    "fmt"
    "net/http"
	"encoding/json"
    "github.com/julienschmidt/httprouter"
)


type SetRequest struct {
	AppName string
	Key     string
	Value   string
	TTL     int64
}

type GetRequest struct {
	AppName string
	Key     string
}


func Index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
    fmt.Fprint(w, "Welcome!\n")
}

func Hello(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
    fmt.Fprintf(w, "hello, %s!\n", ps.ByName("name"))
}


func CacheSet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

}

func CacheGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

}


