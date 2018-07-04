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
}

func NewSiblingsManager() *SiblingsManager {

	sm := &SiblingsManager{
			siblings: make(map[string]Sibling),
			write_ch: make(chan Sibling),
			gc_ch: make(chan bool) }

	go func() {
		for {
			select {

				case s := <-sm.write_ch:
					sm.siblings[s.Node] = s

				case <-sm.gc_ch:
				{
					log.Printf("%+v\n", sm.siblings)

					for k, sibling := range sm.siblings {
						if time.Now().Unix() - sibling.LastCall >= SIBLING_TTL {
							log.Println(k ,"must die")
							delete(sm.siblings, k)
						}

					}
				}
			}

		}
	}()

	return sm
}

func (sm *SiblingsManager) MsgHandler(src *net.UDPAddr, bcount int, bread []byte) {
	log.Println(bcount, "bytes read from", src)
	//log.Println(hex.Dump(bread[:bcount]))

	node := string(bread[:bcount])

	if node != ME {
		s := Sibling{Node: node,
			 Ip: fmt.Sprintf("%s", src.IP),
			 LastCall: time.Now().Unix() }

		sm.write_ch <-s
	}

	sm.gc_ch <-true

}


func (sm *SiblingsManager) PropagateGet(gr *GetRequest) *string {

	var response *string

	response = nil

	scount := len(sm.siblings)

	ch := make(chan *string, scount)

	for k,s := range sm.siblings {
		log.Println("Forwarding get to", k)
		go sm.forwardGet(s, gr, ch)
	}

	for i := 0; i < scount; i++ {

		select {
			case s_response := <-ch:
			{
				log.Printf("Sibling response %+v\n", s_response)
				if s_response != nil {
					response = s_response
				}
			}
		}
	}

	return response
}


func (sm *SiblingsManager) forwardGet(s Sibling, gr *GetRequest, ch chan *string) {

	url := fmt.Sprintf("http://%s:%s/cache/get", s.Ip, MY_PORT)

	gr_json , _ := json.Marshal(gr)

	payload := bytes.NewReader(gr_json)

	req, err := http.NewRequest("POST", url, payload)
	if err != nil {
		log.Println(err)
		ch <-nil
		return
	}

	req.Header.Add("content-type", "application/json")
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
		e := &ExceptionResponse{}
		err = json.Unmarshal(body, e)
		if err != nil {
			log.Println(err, string(body))
		} else {
			log.Printf("%s\n", e)
		}
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

	ch <-&cr.Value

}



