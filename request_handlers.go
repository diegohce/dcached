package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

type setRequest struct {
	SiblingName string `json:"sibling_name,omitempty"`
	AppName     string `json:"appname"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	TTL         int64  `json:"ttl"`
}

type getRequest struct {
	SiblingName string `json:"sibling_name,omitempty"`
	AppName     string `json:"appname"`
	Key         string `json:"key"`
}

type removeRequest struct {
	SiblingName string `json:"sibling_name,omitempty"`
	AppName     string `json:"appname"`
	Key         string `json:"key,omitempty"`
}

type exceptionResponse struct {
	Exception string            `json:"exception"`
	Message   string            `json:"message"`
	Layer     int64             `json:"layer"`
	Extended  map[string]string `json:"extended"`
}

type cacheResponse struct {
	Value     string `json:"value"`
	ElapsedNs int64  `json:"elapsed_ns"`
}

func (c *cacheResponse) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}
func (c *cacheResponse) write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", c)
}

func newException(extype, message string) *exceptionResponse {
	e := &exceptionResponse{
		Exception: extype,
		Message:   message,
		Extended:  map[string]string{},
		Layer:     4}

	return e
}

func (e *exceptionResponse) String() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func (e *exceptionResponse) write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(666)

	fmt.Fprintf(w, "%s", e)
}

func newGetRequest(r *http.Request) (*getRequest, error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	gr := &getRequest{}

	err = json.Unmarshal(body, gr)

	if gr.AppName == "" {
		return nil, fmt.Errorf("appname field not present")

	} else if gr.Key == "" {
		return nil, fmt.Errorf("key field not present")
	}

	return gr, err
}

func newSetRequest(r *http.Request) (*setRequest, error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	sr := &setRequest{}

	err = json.Unmarshal(body, sr)

	if sr.AppName == "" {
		return nil, fmt.Errorf("appname field not present")

	} else if sr.Key == "" {
		return nil, fmt.Errorf("key field not present")

	} else if sr.TTL == 0 {
		return nil, fmt.Errorf("ttl field cannot be zero")

	} else if sr.Value == "" {
		return nil, fmt.Errorf("value field not present")
	}
	return sr, err
}

func newRemoveRequest(r *http.Request) (*removeRequest, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	rr := &removeRequest{}

	err = json.Unmarshal(body, rr)

	return rr, err
}

func cacheSet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	sr, err := newSetRequest(r)
	if err != nil {
		log.Println(err)
		e := newException("InvalidJsonPayloadException", "Invalid json payload")
		e.Extended["more"] = fmt.Sprintf("%s", err)
		e.write(w)
		return
	}

	readop := &readOp{
		done: make(chan bool),
		app:  sr.AppName,
		key:  sr.Key,
	}

	writeop := &writeOp{
		//done: make(chan bool),
		app: sr.AppName,
		key: sr.Key,
		val: sr.Value,
		ttl: sr.TTL,
	}

	log.Printf("request::set checking if %s::%s is mine\n", sr.AppName, sr.Key)

	timeitStart := time.Now().UnixNano()

	mainCache.Reads <- readop
	<-readop.done

	if readop.found {
		log.Printf("request::set %s::%s is mine. Updating storage unit\n", sr.AppName, sr.Key)
		//LOCAL WRITE
		mainCache.Writes <- writeop
		//<-writeop.done

	} else { //KEY IS NOT MINE, ASK SIBLINGS
		log.Printf("request::set %s::%s is not mine\n", sr.AppName, sr.Key)
		if sr.SiblingName == "" { //IF IT ISN'T A FORWARDED SET
			sr.SiblingName = ME
			siblingsMgr.propagateSet(sr)

		} else {
			e := newException("ForwardedKeyNotPresent", "Forwarded key not present in this node")
			e.Extended["sr.SiblingName"] = sr.SiblingName
			e.write(w)
			return
		}
		if sr.SiblingName == ME { //NOBODY HAS THE KEY
			log.Printf("request::set nobody has %s::%s\n", writeop.app, writeop.key)
			//LOCAL WRITE
			mainCache.Writes <- writeop
			//<-writeop.done
			log.Printf("request::set %s::%s stored\n", writeop.app, writeop.key)
		}
	}
	cr := &cacheResponse{
		Value:     sr.Value,
		ElapsedNs: time.Now().UnixNano() - timeitStart,
	}
	cr.write(w)
}

func cacheGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	gr, err := newGetRequest(r)
	if err != nil {
		log.Println(err)
		e := newException("InvalidJsonPayloadException", "Invalid json payload")
		e.Extended["more"] = fmt.Sprintf("%s", err)
		e.write(w)
		return
	}

	readop := &readOp{
		done: make(chan bool),
		app:  gr.AppName,
		key:  gr.Key}

	log.Printf("request::get %+v\n", readop)

	timeitStart := time.Now().UnixNano()

	mainCache.Reads <- readop
	<-readop.done

	log.Printf("request::get %+v done\n", readop)

	if !readop.found {
		if gr.SiblingName == "" { //IF REQUEST IS NOT FROM A SIBLING
			//FORWARD SEARCH TO SIBLINGS
			gr.SiblingName = ME
			sibResponse := siblingsMgr.propagateGet(gr)
			if sibResponse != nil {
				readop.val = *sibResponse
				readop.found = true
				log.Printf("%+v done from sibling %s\n", readop, gr.SiblingName)
			}
		}

		if !readop.found {
			timeitTotal := time.Now().UnixNano() - timeitStart
			e := newException("KeyNotFoundException", "key not found")
			e.Extended["elapsed_ns"] = strconv.Itoa(int(timeitTotal))
			e.write(w)
			return
		}
	}

	timeitTotal := time.Now().UnixNano() - timeitStart

	c := &cacheResponse{
		Value:     readop.val,
		ElapsedNs: timeitTotal,
	}
	c.write(w)

	log.Printf("%+v done\n", readop)

}

func cacheRemove(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	cacheBlock := ps.ByName("cache_block")

	rr, err := newRemoveRequest(r)
	if err != nil {
		log.Println(err)
		e := newException("InvalidJsonPayloadException", "Invalid json payload")
		e.Extended["more"] = fmt.Sprintf("%s", err)
		e.write(w)
		return
	}

	removeop := &removeOp{
		done: make(chan bool),
		app:  rr.AppName,
		key:  rr.Key,
	}

	if cacheBlock == "key" {

		if rr.AppName == "" || rr.Key == "" {
			e := newException("InvalidJsonPayloadException", "Invalid json payload")
			e.Extended["more"] = "appname and key must be present"
			e.write(w)
			return
		}

		mainCache.RemoveKey <- removeop
		<-removeop.done

		if !removeop.found && rr.SiblingName == "" {
			//PROPAGATE REMOVE KEY
			rr.SiblingName = ME
			siblingsMgr.propagateRemove(rr, cacheBlock)
		}

	} else if cacheBlock == "application" {

		if rr.AppName == "" {
			e := newException("InvalidJsonPayloadException", "Invalid json payload")
			e.Extended["more"] = "appname must be present"
			e.write(w)
			return
		}

		mainCache.RemoveApp <- removeop
		<-removeop.done

		if rr.SiblingName == "" {
			//PROPAGATE REMOVE APP
			rr.SiblingName = ME
			siblingsMgr.propagateRemove(rr, cacheBlock)
		}

	} else if cacheBlock == "all" {

		if rr.AppName == "" {
			e := newException("InvalidJsonPayloadException", "Invalid json payload")
			e.Extended["more"] = "appname must be present"
			e.write(w)
			return
		}

		mainCache.RemoveAll <- removeop
		<-removeop.done

		if rr.SiblingName == "" {
			//PROPAGATE REMOVE APP
			rr.SiblingName = ME
			siblingsMgr.propagateRemove(rr, cacheBlock)
		}

	} else {
		e := newException("ResourceNotFoundException", "Resource not found")
		e.write(w)
	}

}
