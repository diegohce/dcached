package main

import (
    "fmt"
	"log"
	//"time"
	//"strconv"
    "net/http"
	"encoding/json"
	"io/ioutil"
    "github.com/julienschmidt/httprouter"
)


func newExportUnit(r *http.Request) (*ExportUnit, error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	eu := &ExportUnit{}

	err = json.Unmarshal(body, eu)

	return eu, err
}

func CacheImport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	eu, err := newExportUnit(r)
	if err != nil {
		log.Println(err)
		e := NewException("InvalidJsonPayloadException", "Invalid json payload")
		e.Extended["more"] = fmt.Sprintf("%s", err)
		e.Write(w)
		return
	}

	importop := &writeOp {
		done: make(chan bool),
		app: eu.AppName,
		key: eu.Key,
		val: eu.Value,
		ttl: eu.TTL,
		ct: eu.CreatedAt,
	 }

	//timeit_start := time.Now().UnixNano()

	//LOCAL WRITE
	CACHE.Imports <-importop
	//<-importop.done //CAN THIS LINE BE REMOVED FOR PERFORMANCE?
}

