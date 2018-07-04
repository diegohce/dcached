package main

import (
    "fmt"
	"log"
    "net/http"
	"encoding/json"
	//"io"
	"io/ioutil"
    "github.com/julienschmidt/httprouter"
)


type SetRequest struct {
	AppName string `json:"appname"`
	Key     string `json:"key"`
	Value   string `json:"value"`
	TTL     int64  `json:"ttl"`
}

type GetRequest struct {
	AppName string `json:"appname"`
	Key     string `json:"key"`
}


type ExceptionResponse struct {
	Exception string   `json:"exception"`
	Message   string   `json:"message"`
	Layer     int64    `json:"layer"`
	Extended  map[string]string `json:"extended"`
}

type CacheResponse struct {
	Value string `json:"value"`
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



func makeGetRequest(r *http.Request) (*GetRequest, error) {

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

func makeSetRequest(r *http.Request) (*SetRequest, error) {

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


func CacheSet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	sr, err := makeSetRequest(r)
	if err != nil {
		log.Println(err)
		e := NewException("InvalidJsonPayloadException", "Invalid json payload")
		e.Extended["more"] = fmt.Sprintf("%s", err)
		e.Write(w)
		return
	}

	writeop := &writeOp {
				done: make(chan bool),
				app: sr.AppName,
				key: sr.Key,
				val: sr.Value,
				ttl: sr.TTL }

	log.Printf("%+v\n", writeop)

	//LOCAL WRITE
	CACHE.Writes <-writeop
	<-writeop.done

	log.Printf("%+v done\n", writeop)
}

func CacheGet(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	gr, err := makeGetRequest(r)
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

	log.Printf("%+v\n", readop)

	CACHE.Reads <-readop
	<-readop.done

	if !readop.found {
		//FORWARD SEARCH TO SIBLINGS
		sib_response := SIBLINGS_MANAGER.PropagateGet(gr)
		if sib_response != nil {
			c := &CacheResponse{Value: *sib_response}
			c.Write(w)
			log.Printf("%+v done from sibling\n", readop)
			return
		}
		e := NewException("KeyNotFoundException", "key not found")
		e.Write(w)
		return
	}

	c := &CacheResponse{Value: readop.val}
	c.Write(w)

	log.Printf("%+v done\n", readop)

}


