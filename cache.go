package main

import (
	//"fmt"
	"log"
	"time"
)

type readOp struct {
	app   string
	key   string
	val   string
	found bool
	done  chan bool
}

type writeOp struct {
	app  string
	key  string
	val  string
	ttl  int64
	ct   int64
	done chan bool
}

type removeOp struct {
	app   string
	key   string
	found bool
	done  chan bool
}

type statsOp struct {
	stats *cacheStats
	done  chan bool
}

type storageUnit struct {
	CreatedAt int64
	Value     string
	TTL       int64
}

type cacheStats struct {
	Node         string    `json:"node"`
	Nodes        []sibling `json:"nodes"`
	Applications []string  `json:"applications"`
	MemBytes     int64     `json:"mem_bytes"`
	ElapsedNs    int64     `json:"elapsed_ns"`
}

type appCache map[string]map[string]storageUnit

type cacheOperator struct {
	storage    appCache
	Reads      chan *readOp
	Writes     chan *writeOp
	RemoveKey  chan *removeOp
	RemoveApp  chan *removeOp
	RemoveAll  chan *removeOp
	CacheStats chan *statsOp
	Imports    chan *writeOp
	GcTimer    *time.Ticker
}

/*func init() {
	CACHE = NewCache()
}
*/

func newCache() *cacheOperator {

	c := &cacheOperator{storage: appCache{},
		Reads:      make(chan *readOp),
		Writes:     make(chan *writeOp),
		RemoveKey:  make(chan *removeOp),
		RemoveApp:  make(chan *removeOp),
		RemoveAll:  make(chan *removeOp),
		CacheStats: make(chan *statsOp),
		Imports:    make(chan *writeOp),
		GcTimer:    time.NewTicker(time.Duration(cacheGCFreq) * time.Second),
	}

	go func() {
		for {
			select {
			case read := <-c.Reads:
				{
					log.Println("cache::readop", read)
					read.val, read.found = c.get(read.app, read.key)
					read.done <- true
					log.Println("cache::readop", read, "done")
				}
			case write := <-c.Writes:
				{
					c.set(write.app, write.key, write.val, write.ttl)
					//write.done <- true
				}
			case removek := <-c.RemoveKey:
				{
					removek.found = c.removeKey(removek.app, removek.key)
					removek.done <- true
				}
			case removea := <-c.RemoveApp:
				{
					removea.found = c.removeApp(removea.app)
					removea.done <- true
				}
			case removeall := <-c.RemoveAll:
				{
					c.removeAll()
					removeall.done <- true
				}
			case stats := <-c.CacheStats:
				{
					stats.stats = c.stats()
					stats.done <- true
				}
			case importf := <-c.Imports:
				{
					c.importForeign(importf)
					//importf.done <- true
				}
			case <-c.GcTimer.C:
				c.gc()
			}
		}
	}()

	return c
}

func (c *cacheOperator) set(appname, key, value string, ttl int64) {

	su := storageUnit{
		CreatedAt: time.Now().Unix(),
		Value:     value,
		TTL:       ttl}

	_, ok := c.storage[appname]
	if !ok {
		c.storage[appname] = map[string]storageUnit{}
	}

	c.storage[appname][key] = su

}

func (c *cacheOperator) get(appname, key string) (string, bool) {

	cache, ok := c.storage[appname]
	if !ok {
		return "", ok
	}

	su, ok := cache[key]
	if !ok {
		return "", ok
	}

	if time.Now().Unix()-su.CreatedAt >= su.TTL {
		delete(c.storage[appname], key)
		return "", false
	}

	return su.Value, ok
}

func (c *cacheOperator) removeKey(appname, key string) bool {

	cache, ok := c.storage[appname]
	if !ok {
		return false
	}

	_, ok = cache[key]
	if !ok {
		return false
	}

	delete(c.storage[appname], key)
	return true
}

func (c *cacheOperator) removeApp(appname string) bool {

	_, ok := c.storage[appname]
	if !ok {
		return false
	}

	delete(c.storage, appname)
	return true
}

func (c *cacheOperator) removeAll() {

	//for app, _ := range c.storage {
	for app := range c.storage {
		delete(c.storage, app)
	}
}

func (c *cacheOperator) gc() {

	log.Println("cache::Starting GC")

	for app, cache := range c.storage {
		for k, su := range cache {
			if time.Now().Unix()-su.CreatedAt > su.TTL {
				delete(c.storage[app], k)
				log.Printf("cache::GC %s->%s has been killed", app, k)
			}
		}
		if len(c.storage[app]) == 0 {
			delete(c.storage, app)
		}
	}
	log.Println("cache::GC finished")
}

func (c *cacheOperator) stats() *cacheStats {

	cs := &cacheStats{
		Node:     ME,
		Nodes:    siblingsMgr.getSiblings(),
		MemBytes: 0,
	}

	for app, cache := range c.storage {
		cs.Applications = append(cs.Applications, app)
		for k, su := range cache {
			cs.MemBytes += int64(len(k)) + int64(len(su.Value))

		}
	}

	return cs
}

type exportUnit struct {
	AppName   string `json:"appname"`
	CreatedAt int64  `json:"created_at"`
	Key       string `json:"key"`
	Value     string `json:"val"`
	TTL       int64  `json:"ttl"`
}

func (c *cacheOperator) contentExporter(ch chan *exportUnit) {

	for app, cache := range c.storage {
		for k, su := range cache {
			eu := &exportUnit{
				AppName:   app,
				CreatedAt: su.CreatedAt,
				Key:       k,
				Value:     su.Value,
				TTL:       su.TTL,
			}
			delete(c.storage[app], k)
			ch <- eu
			log.Printf("cache::export %+v ready to export\n", eu)
		}
	}
	close(ch)
}

func (c *cacheOperator) importForeign(wop *writeOp) {
	su := storageUnit{
		CreatedAt: wop.ct,
		Value:     wop.val,
		TTL:       wop.ttl,
	}

	_, ok := c.storage[wop.app]
	if !ok {
		c.storage[wop.app] = map[string]storageUnit{}
	}

	c.storage[wop.app][wop.key] = su
}
