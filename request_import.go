package main

import (
	"fmt"
	"log"

	//"time"
	//"strconv"
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func newExportUnit(r *http.Request) (*exportUnit, error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	eu := &exportUnit{}

	err = json.Unmarshal(body, eu)

	return eu, err
}

func cacheImport(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	eu, err := newExportUnit(r)
	if err != nil {
		log.Println(err)
		e := newException("InvalidJsonPayloadException", "Invalid json payload")
		e.Extended["more"] = fmt.Sprintf("%s", err)
		e.write(w)
		return
	}

	importop := &writeOp{
		done: make(chan bool),
		app:  eu.AppName,
		key:  eu.Key,
		val:  eu.Value,
		ttl:  eu.TTL,
		ct:   eu.CreatedAt,
	}

	//timeit_start := time.Now().UnixNano()

	//LOCAL WRITE
	mainCache.Imports <- importop
	//<-importop.done //CAN THIS LINE BE REMOVED FOR PERFORMANCE?
}
