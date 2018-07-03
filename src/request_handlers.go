package main

import (
    "fmt"
    "net/http"
	"encoding/json"
	"io"
	"io/ioutil"
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



func makeGetRequest(r *http.Request) (*GetRequest, error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	gr := &GetRequest{}

	err := json.Unmarshal(body, gr)

	return gr, err
}

func makeSetRequest(r *http.Request) (*SetRequest, error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	sr := &SetRequest{}

	err := json.Unmarshal(body, sr)

	return sr, err
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


