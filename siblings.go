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
	Node     string `json:"node"`
	Ip       string `json:"ip"`
	LastCall int64  `json:"last_call"`
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

func (sm *SiblingsManager) GetSiblings() []Sibling  {

	var siblings []Sibling

	sm.mutex.RLock()
	for _, sibling := range sm.siblings {
		siblings = append(siblings, sibling)
	}
	sm.mutex.RUnlock()

	return siblings
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


func (sm *SiblingsManager) PropagateSet(sr *SetRequest) {

	sm.mutex.RLock()

	scount := len(sm.siblings)
	ch := make(chan int, scount)

	for k,s := range sm.siblings {
		log.Println("siblings::Forwarding set to", k)
		go sm.forwardSet(s, sr, ch)
	}
	sm.mutex.RUnlock()

	for i := 0; i < scount; i++ {
		status_code := <-ch

		if status_code == 200 {
			log.Printf("siblings::Sibling::set response %d\n", status_code)
			break
		} else {
			log.Println("siblings::Sibling::set response", status_code)
		}
	}
}


func (sm *SiblingsManager) forwardSet(s Sibling, sr *SetRequest, ch chan int) {

	url := fmt.Sprintf("http://%s:%s/%s", s.Ip, CACHE_PORT, CACHE_SET_URL)

	sr_json , _ := json.Marshal(sr)

	payload := bytes.NewReader(sr_json)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Println(err)
		ch <-0
		return
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		ch <-0
		return
	}
	defer res.Body.Close()


	if res.StatusCode != 200 {
		log.Printf("Sibling %s does not have %s::%s\n", s.Node, sr.AppName, sr.Key)
	} else {
		sr.SiblingName = s.Node
		log.Printf("Sibling %s does have %s::%s\n", s.Node, sr.AppName, sr.Key)
	}

	ch <-res.StatusCode

}



func (sm *SiblingsManager) PropagateRemove(rr *RemoveRequest, cache_block string) {

	sm.mutex.RLock()

	for k,s := range sm.siblings {
		log.Println("siblings::Forwarding remove to", k)
		go sm.forwardRemove(s, rr, cache_block)
	}
	sm.mutex.RUnlock()

}


func (sm *SiblingsManager) forwardRemove(s Sibling, rr *RemoveRequest, cache_block string) {

	var url string

	if cache_block == "key" {
		url = fmt.Sprintf("http://%s:%s/%s", s.Ip, CACHE_PORT, CACHE_REMOVE_KEY_URL)

	} else if cache_block == "application" {
		url = fmt.Sprintf("http://%s:%s/%s", s.Ip, CACHE_PORT, CACHE_REMOVE_APP_URL)

	} else if cache_block == "all" {
		url = fmt.Sprintf("http://%s:%s/%s", s.Ip, CACHE_PORT, CACHE_REMOVE_ALL_URL)

	} else {
		return
	}

	rr_json , _ := json.Marshal(rr)

	payload := bytes.NewReader(rr_json)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()

}

func (sm *SiblingsManager) PropagateStats() chan *CacheStats {

	sm.mutex.RLock()

	scount := len(sm.siblings)
	ch := make(chan *CacheStats, scount)

	for k ,s := range sm.siblings {
		log.Println("siblings::Forwarding statsop to", k)
		go sm.forwardStats(s, ch)
	}
	sm.mutex.RUnlock()

	if scount == 0 {
		close(ch)
		return ch
	}

	out_ch := make(chan *CacheStats, scount)

	for i := 0; i < scount; i++ {
		stat := <-ch
		out_ch <-stat
	}
	close(out_ch)

	return out_ch
}


func (sm *SiblingsManager) forwardStats(s Sibling, ch chan *CacheStats ) {

	url := fmt.Sprintf("http://%s:%s/%s", s.Ip, CACHE_PORT, CACHE_STATS_LOCAL_URL)


	req, err := http.NewRequest("GET", url, nil)
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
		log.Println(err, string(body))
		ch<-nil
		return
	}

	cs := &CacheStats{}

	err = json.Unmarshal(body, cs)
	if err != nil {
		log.Println(err)
		ch <-nil
		return
	}

	log.Printf("Sending %+v to stats channel\n", *cs)
	ch <-cs
}

func (sm *SiblingsManager) DistributeContent() {

	var sib_list []Sibling

	sm.mutex.RLock()
	scount := len(sm.siblings)
	if scount == 0 {
		sm.mutex.RUnlock()
		return
	}

	for _ ,s := range sm.siblings {
		sib_list = append(sib_list, s)
	}
	sm.mutex.RUnlock()

	ch := make(chan *ExportUnit)

	go CACHE.contentExporter(ch)

	sib_idx := 0
	sib_list_len := len(sib_list)

	for eu := range ch {
		if sib_idx == sib_list_len {
			sib_idx = 0
		}
		sib := sib_list[sib_idx]
		sib_idx++
		log.Printf("siblings::exporting %+v to %s\n", eu, sib.Node)
		sm.distributeExport(sib, eu)

	}

}

func (sm *SiblingsManager) distributeExport(s Sibling, eu *ExportUnit) {

	url := fmt.Sprintf("http://%s:%s/%s", s.Ip, CACHE_PORT, CACHE_IMPORT_URL)

	eu_json , _ := json.Marshal(eu)

	payload := bytes.NewReader(eu_json)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Println(err)
		return
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return
	}
	defer res.Body.Close()


	log.Printf("Sibling %s status code %d\n", s.Node, res.StatusCode )

}



