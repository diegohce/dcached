package main

import (
	"fmt"
	"log"
	"time"
	"bytes"
	"net"
	"net/http"
	"io/ioutil"
	"encoding/json"
	//"encoding/hex"
	"sync"
)


type Sibling struct {
	Node     string
	Ip       string
	LastCall int64
}


type SiblingsManager struct {
	siblings map[string]Sibling
	write_ch chan Sibling
	gc_ch    chan bool
	mutex    sync.RWMutex
}

func NewSiblingsManager() *SiblingsManager {

	sm := &SiblingsManager{
		siblings: make(map[string]Sibling),
		write_ch: make(chan Sibling),
		mutex: sync.RWMutex{},
	}

	go func() {
		for {
			s := <-sm.write_ch

			sm.mutex.Lock()

			_, ok := sm.siblings[s.Node]
			sm.siblings[s.Node] = s
			if !ok {
				log.Printf("siblings::add::count %+v\n", len(sm.siblings))
			}

			sm.mutex.Unlock()
		}
	}()

	return sm
}

func (sm *SiblingsManager) MsgHandler(src *net.UDPAddr, bcount int, bread []byte) {
	//log.Println(bcount, "bytes read from", src)
	//log.Println(hex.Dump(bread[:bcount]))

	node := string(bread[:bcount])

	if node != ME {
		s := Sibling{Node: node,
			 Ip: fmt.Sprintf("%s", src.IP),
			 LastCall: time.Now().Unix() }
		sm.write_ch <-s
	}

	sm.mutex.Lock()
	for k, sibling := range sm.siblings {
		if time.Now().Unix() - sibling.LastCall >= SIBLING_TTL {
			log.Println("siblings::gc", k ,"must die")
			delete(sm.siblings, k)
			log.Printf("siblings::gc::count %+v\n", len(sm.siblings))
		}
	}
	sm.mutex.Unlock()
}

func (sm *SiblingsManager) GetSibling(node string) *Sibling {

	sm.mutex.RLock()
	s, ok := sm.siblings[node]
	sm.mutex.RUnlock()
	if !ok {
		return nil
	}
	return &s
}

func (sm *SiblingsManager) PropagateGet(gr *GetRequest) *string {

	var response *string

	response = nil


	sm.mutex.RLock()

	scount := len(sm.siblings)
	ch := make(chan *string, scount)

	for k,s := range sm.siblings {
		log.Println("siblings::Forwarding get to", k)
		go sm.forwardGet(s, gr, ch)
	}
	sm.mutex.RUnlock()

	for i := 0; i < scount; i++ {
		s_response := <-ch

		if s_response != nil {
			log.Printf("siblings::Sibling response %+v\n", *s_response)
			response = s_response
			break //???? REVISAR QUE PASA CON EL RESTO DE LAS GOROUTINES CUANDO SE SALE DE ACA Y LAS DEMAS NO TERMINARON
		} else {
			log.Println("siblings::Sibling response nil")
		}
	}

	return response
}


func (sm *SiblingsManager) forwardGet(s Sibling, gr *GetRequest, ch chan *string) {

	url := fmt.Sprintf("http://%s:%s/%s", s.Ip, CACHE_PORT, CACHE_GET_URL)

	gr_json , _ := json.Marshal(gr)

	payload := bytes.NewReader(gr_json)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Println(err)
		ch <-nil
		return
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		ch <-nil
		return
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != 200 {
		//e := &ExceptionResponse{}
		//err = json.Unmarshal(body, e)
		//if err != nil {
			log.Println(err, string(body))
		//} else {
		//	log.Printf("%s\n", e)
		//}
		ch<-nil
		return
	}

	cr := &CacheResponse{}

	err = json.Unmarshal(body, &cr)
	if err != nil {
		log.Println(err)
		ch <-nil
		return
	}

	gr.SiblingName = s.Node
	ch <-&cr.Value

}



