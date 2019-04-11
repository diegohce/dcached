package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	//"encoding/hex"
	"sync"
)

type sibling struct {
	Node     string `json:"node"`
	IPAddr   string `json:"ip"`
	LastCall int64  `json:"last_call"`
}

type siblingsManager struct {
	siblings map[string]sibling
	writeCh  chan sibling
	gcCh     chan bool
	mutex    sync.RWMutex
}

func newSiblingsManager() *siblingsManager {

	sm := &siblingsManager{
		siblings: make(map[string]sibling),
		writeCh:  make(chan sibling),
		mutex:    sync.RWMutex{},
	}

	go func() {
		for {
			s := <-sm.writeCh

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

func (sm *siblingsManager) msgHandler(src *net.UDPAddr, bcount int, bread []byte) {
	//log.Println(bcount, "bytes read from", src)
	//log.Println(hex.Dump(bread[:bcount]))

	node := string(bread[:bcount])

	if node != ME {
		s := sibling{Node: node,
			IPAddr:   fmt.Sprintf("%s", src.IP),
			LastCall: time.Now().Unix()}
		sm.writeCh <- s
	}

	sm.mutex.Lock()
	for k, sibling := range sm.siblings {
		if time.Now().Unix()-sibling.LastCall >= siblingTTL {
			log.Println("siblings::gc", k, "must die")
			delete(sm.siblings, k)
			log.Printf("siblings::gc::count %+v\n", len(sm.siblings))
		}
	}
	sm.mutex.Unlock()
}

func (sm *siblingsManager) getSiblings() []sibling {

	var siblings []sibling

	sm.mutex.RLock()
	for _, sibling := range sm.siblings {
		siblings = append(siblings, sibling)
	}
	sm.mutex.RUnlock()

	return siblings
}

func (sm *siblingsManager) propagateGet(gr *getRequest) *string {

	var response *string

	response = nil

	sm.mutex.RLock()

	scount := len(sm.siblings)
	ch := make(chan *string, scount)

	for k, s := range sm.siblings {
		log.Println("siblings::Forwarding get to", k)
		go sm.forwardGet(s, gr, ch)
	}
	sm.mutex.RUnlock()

	for i := 0; i < scount; i++ {
		sResponse := <-ch

		if sResponse != nil {
			log.Printf("siblings::Sibling response %+v\n", *sResponse)
			response = sResponse
			break //???? REVISAR QUE PASA CON EL RESTO DE LAS GOROUTINES CUANDO SE SALE DE ACA Y LAS DEMAS NO TERMINARON
		} else {
			log.Println("siblings::Sibling response nil")
		}
	}

	return response
}

func (sm *siblingsManager) forwardGet(s sibling, gr *getRequest, ch chan *string) {

	url := fmt.Sprintf("http://%s:%s/%s", s.IPAddr, cachePort, cacheGetURL)

	grJSON, _ := json.Marshal(gr)

	payload := bytes.NewReader(grJSON)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Println(err)
		ch <- nil
		return
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		ch <- nil
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
		ch <- nil
		return
	}

	cr := &cacheResponse{}

	err = json.Unmarshal(body, &cr)
	if err != nil {
		log.Println(err)
		ch <- nil
		return
	}

	gr.SiblingName = s.Node
	ch <- &cr.Value

}

func (sm *siblingsManager) propagateSet(sr *setRequest) {

	sm.mutex.RLock()

	scount := len(sm.siblings)
	ch := make(chan int, scount)

	for k, s := range sm.siblings {
		log.Println("siblings::Forwarding set to", k)
		go sm.forwardSet(s, sr, ch)
	}
	sm.mutex.RUnlock()

	for i := 0; i < scount; i++ {
		statusCode := <-ch

		if statusCode == 200 {
			log.Printf("siblings::Sibling::set response %d\n", statusCode)
			break
		} else {
			log.Println("siblings::Sibling::set response", statusCode)
		}
	}
}

func (sm *siblingsManager) forwardSet(s sibling, sr *setRequest, ch chan int) {

	url := fmt.Sprintf("http://%s:%s/%s", s.IPAddr, cachePort, cacheSetURL)

	srJSON, _ := json.Marshal(sr)

	payload := bytes.NewReader(srJSON)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Println(err)
		ch <- 0
		return
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		ch <- 0
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		log.Printf("Sibling %s does not have %s::%s\n", s.Node, sr.AppName, sr.Key)
	} else {
		sr.SiblingName = s.Node
		log.Printf("Sibling %s does have %s::%s\n", s.Node, sr.AppName, sr.Key)
	}

	ch <- res.StatusCode

}

func (sm *siblingsManager) propagateRemove(rr *removeRequest, cacheBlock string) {

	sm.mutex.RLock()

	for k, s := range sm.siblings {
		log.Println("siblings::Forwarding remove to", k)
		go sm.forwardRemove(s, rr, cacheBlock)
	}
	sm.mutex.RUnlock()

}

func (sm *siblingsManager) forwardRemove(s sibling, rr *removeRequest, cacheBlock string) {

	var url string

	if cacheBlock == "key" {
		url = fmt.Sprintf("http://%s:%s/%s", s.IPAddr, cachePort, cacheRemoveKeyURL)

	} else if cacheBlock == "application" {
		url = fmt.Sprintf("http://%s:%s/%s", s.IPAddr, cachePort, cacheRemoveAppURL)

	} else if cacheBlock == "all" {
		url = fmt.Sprintf("http://%s:%s/%s", s.IPAddr, cachePort, cacheRemoveAllURL)

	} else {
		return
	}

	rrJSON, _ := json.Marshal(rr)

	payload := bytes.NewReader(rrJSON)

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

func (sm *siblingsManager) propagateStats() chan *cacheStats {

	sm.mutex.RLock()

	scount := len(sm.siblings)
	ch := make(chan *cacheStats, scount)

	for k, s := range sm.siblings {
		log.Println("siblings::Forwarding statsop to", k)
		go sm.forwardStats(s, ch)
	}
	sm.mutex.RUnlock()

	if scount == 0 {
		close(ch)
		return ch
	}

	outCh := make(chan *cacheStats, scount)

	for i := 0; i < scount; i++ {
		stat := <-ch
		outCh <- stat
	}
	close(outCh)

	return outCh
}

func (sm *siblingsManager) forwardStats(s sibling, ch chan *cacheStats) {

	url := fmt.Sprintf("http://%s:%s/%s", s.IPAddr, cachePort, cacheStatslocalURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println(err)
		ch <- nil
		return
	}

	req.Header.Add("Content-Type", "application/json")
	client := &http.Client{}

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		ch <- nil
		return
	}
	defer res.Body.Close()

	body, _ := ioutil.ReadAll(res.Body)

	if res.StatusCode != 200 {
		log.Println(err, string(body))
		ch <- nil
		return
	}

	cs := &cacheStats{}

	err = json.Unmarshal(body, cs)
	if err != nil {
		log.Println(err)
		ch <- nil
		return
	}

	log.Printf("Sending %+v to stats channel\n", *cs)
	ch <- cs
}

func (sm *siblingsManager) distributeContent() {

	var sibList []sibling

	sm.mutex.RLock()
	scount := len(sm.siblings)
	if scount == 0 {
		sm.mutex.RUnlock()
		return
	}

	for _, s := range sm.siblings {
		sibList = append(sibList, s)
	}
	sm.mutex.RUnlock()

	ch := make(chan *exportUnit)

	go mainCache.contentExporter(ch)

	sibIdx := 0
	sibListLen := len(sibList)

	for eu := range ch {
		if sibIdx == sibListLen {
			sibIdx = 0
		}
		sib := sibList[sibIdx]
		sibIdx++
		log.Printf("siblings::exporting %+v to %s\n", eu, sib.Node)
		sm.distributeExport(sib, eu)

	}

}

func (sm *siblingsManager) distributeExport(s sibling, eu *exportUnit) {

	url := fmt.Sprintf("http://%s:%s/%s", s.IPAddr, cachePort, cacheImportURL)

	euJSON, _ := json.Marshal(eu)

	payload := bytes.NewReader(euJSON)

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

	log.Printf("Sibling %s status code %d\n", s.Node, res.StatusCode)

}
