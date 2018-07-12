package main

import (
    "fmt"
	"log"
	"time"
	"strconv"
    "net/http"
	"encoding/json"
	"io/ioutil"
    "github.com/julienschmidt/httprouter"
)


type SetRequest struct {
	SiblingName string `json:"sibling_name,omitempty"`
	AppName     string `json:"appname"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	TTL         int64  `json:"ttl"`
}

type GetRequest struct {
	SiblingName string `json:"sibling_name,omitempty"`
	AppName     string `json:"appname"`
	Key         string `json:"key"`
}

type RemoveRequest struct {
	SiblingName string `json:"sibling_name,omitempty"`
	AppName     string `json:"appname"`
	Key         string `json:"key,omitempty"`
}

type ExceptionResponse struct {
	Exception string   `json:"exception"`
	Message   string   `json:"message"`
	Layer     int64    `json:"layer"`
	Extended  map[string]string `json:"extended"`
}

type CacheResponse struct {
	Value      string `json:"value"`
	Elapsed_ns int64 `json:"elapsed_ns"`
}
func (c *CacheResponse) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}
func (c *CacheResponse) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	fmt.Fprintf(w, "%s", c)
}


func NewException(extype, message string) *ExceptionResponse {
	e := &ExceptionResponse {
		Exception: extype,
		Message  : message,
		Extended : map[string]string{},
		Layer    : 4}

	return e
}

func (e *ExceptionResponse) String() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func (e *ExceptionResponse) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(400)
	fmt.Fprintf(w, "%s", e)
}



func newGetRequest(r *http.Request) (*GetRequest, error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	gr := &GetRequest{}

	err = json.Unmarshal(body, gr)

	if gr.AppName == "" {
		return nil, fmt.Errorf("appname field not present")

	} else if gr.Key == "" {
		return nil, fmt.Errorf("key field not present")
	}

	return gr, err
}

func newSetRequest(r *http.Request) (*SetRequest, error) {

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	sr := &SetRequest{}

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

func newRemoveRequest(r *http.Request) (*RemoveRequest, error) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	rr := &RemoveRequest{}

	err = json.Unmarshal(body, rr)

	return rr, err
}


func CacheSet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	sr, err := newSetRequest(r)
	if err != nil {
		log.Println(err)
		e := NewException("InvalidJsonPayloadException", "Invalid json payload")
		e.Extended["more"] = fmt.Sprintf("%s", err)
		e.Write(w)
		return
	}

	readop := &readOp {
				done: make(chan bool),
				app: sr.AppName,
				key: sr.Key }

	writeop := &writeOp {
				done: make(chan bool),
				app: sr.AppName,
				key: sr.Key,
				val: sr.Value,
				ttl: sr.TTL }

	log.Printf("request::set checking if %s::%s is mine\n", sr.AppName, sr.Key)

	timeit_start := time.Now().UnixNano()

	CACHE.Reads <-readop
	<-readop.done

	if readop.found {
		log.Printf("request::set %s::%s is mine. Updating storage unit\n", sr.AppName, sr.Key)
		//LOCAL WRITE
		CACHE.Writes <-writeop
		<-writeop.done

	} else { //KEY IS NOT MINE, ASK SIBLINGS
		log.Printf("request::set %s::%s is not mine\n", sr.AppName, sr.Key)
		if sr.SiblingName == "" { //IF IT ISN'T A FORWARDED SET
			sr.SiblingName = ME
			SIBLINGS_MANAGER.PropagateSet(sr)

		} else {
			e := NewException("ForwardedKeyNotPresent", "Forwarded key not present in this node")
			e.Extended["sr.SiblingName"] = sr.SiblingName
			e.Write(w)
			return
		}
		if sr.SiblingName == ME { //NOBODY HAS THE KEY
			log.Printf("request::set nobody has %s::%s\n", writeop.app, writeop.key)
			//LOCAL WRITE
			CACHE.Writes <-writeop
			<-writeop.done
			log.Printf("request::set %s::%s stored\n", writeop.app, writeop.key)
		}
	}
	cr := &CacheResponse{
		Value: sr.Value,
		Elapsed_ns: time.Now().UnixNano() - timeit_start,
	}
	cr.Write(w)
}

func CacheGet(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	gr, err := newGetRequest(r)
	if err != nil {
		log.Println(err)
		e := NewException("InvalidJsonPayloadException", "Invalid json payload")
		e.Extended["more"] = fmt.Sprintf("%s", err)
		e.Write(w)
		return
	}

	readop := &readOp {
				done: make(chan bool),
				app: gr.AppName,
				key: gr.Key }

	log.Printf("request::get %+v\n", readop)

	timeit_start := time.Now().UnixNano()


	CACHE.Reads <-readop
	<-readop.done

	log.Printf("request::get %+v done\n", readop)

	if !readop.found  {
		if gr.SiblingName == "" { //IF REQUEST IS NOT FROM A SIBLING
			//FORWARD SEARCH TO SIBLINGS
			gr.SiblingName = ME
			sib_response := SIBLINGS_MANAGER.PropagateGet(gr)
			if sib_response != nil {
				readop.val = *sib_response
				readop.found = true
				log.Printf("%+v done from sibling %s\n", readop, gr.SiblingName)
			}
		}

		if !readop.found {
			timeit_total := time.Now().UnixNano() - timeit_start
			e := NewException("KeyNotFoundException", "key not found")
			e.Extended["elapsed_ns"] = strconv.Itoa(int(timeit_total))
			e.Write(w)
			return
		}
	}

	timeit_total := time.Now().UnixNano() - timeit_start

	c := &CacheResponse{
		Value: readop.val,
		Elapsed_ns: timeit_total,
	}
	c.Write(w)

	log.Printf("%+v done\n", readop)

}

func CacheRemove(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	cache_block := ps.ByName("cache_block")

	rr, err := newRemoveRequest(r)
	if err != nil {
		log.Println(err)
		e := NewException("InvalidJsonPayloadException", "Invalid json payload")
		e.Extended["more"] = fmt.Sprintf("%s", err)
		e.Write(w)
		return
	}

	removeop := &removeOp {
		done: make(chan bool),
		app: rr.AppName,
		key: rr.Key,
	}

	if cache_block == "key" {

		if rr.AppName == "" || rr.Key == "" {
			e := NewException("InvalidJsonPayloadException", "Invalid json payload")
			e.Extended["more"] = "appname and key must be present"
			e.Write(w)
			return
		}

		CACHE.RemoveKey <-removeop
		<-removeop.done

		if !removeop.found && rr.SiblingName == "" {
			//PROPAGATE REMOVE KEY
			rr.SiblingName = ME
			SIBLINGS_MANAGER.PropagateRemove(rr, cache_block)
		}

	} else if cache_block == "application" {

		if rr.AppName == ""  {
			e := NewException("InvalidJsonPayloadException", "Invalid json payload")
			e.Extended["more"] = "appname must be present"
			e.Write(w)
			return
		}

		CACHE.RemoveApp <-removeop
		<-removeop.done

		if rr.SiblingName == "" {
			//PROPAGATE REMOVE APP
			rr.SiblingName = ME
			SIBLINGS_MANAGER.PropagateRemove(rr, cache_block)
		}

	} else if cache_block == "all" {

		if rr.AppName == ""  {
			e := NewException("InvalidJsonPayloadException", "Invalid json payload")
			e.Extended["more"] = "appname must be present"
			e.Write(w)
			return
		}

		CACHE.RemoveAll <-removeop
		<-removeop.done

		if rr.SiblingName == "" {
			//PROPAGATE REMOVE APP
			rr.SiblingName = ME
			SIBLINGS_MANAGER.PropagateRemove(rr, cache_block)
		}

	} else {
		e := NewException("ResourceNotFoundException", "Resource not found")
		e.Write(w)
	}

}


