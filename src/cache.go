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
	stats *CacheStats
	done  chan bool
}

type StorageUnit struct {
	CreatedAt int64
	Value     string
	TTL       int64
}

type CacheStats struct {
	Node         string    `json:"node"`
	Nodes        []Sibling `json:"nodes"`
	Applications []string  `json:"applications"`
	MemBytes     int64     `json:"mem_bytes"`
	Elapsed_ns   int64     `json:"elapsed_ns"`
}

type AppCache map[string]map[string]StorageUnit

type Cache struct {
	storage   AppCache
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

func NewCache() *Cache {

	c := &Cache{storage: AppCache{},
		Reads: make(chan *readOp),
        Writes: make(chan *writeOp),
		RemoveKey: make(chan *removeOp),
		RemoveApp: make(chan *removeOp),
		RemoveAll: make(chan *removeOp),
		CacheStats: make(chan *statsOp),
        Imports: make(chan *writeOp),
		GcTimer: time.NewTicker(time.Duration(CACHE_GC_FREQ) * time.Second),
	 }

	go func(){
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
					write.done <- true
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


func (c *Cache) set(appname, key, value string, ttl int64) {

	su := StorageUnit {
		CreatedAt: time.Now().Unix(),
		Value: value,
		TTL: ttl }

	_, ok := c.storage[appname]
	if !ok {
		c.storage[appname] = map[string]StorageUnit{}
	}

	c.storage[appname][key] = su

}

func (c *Cache) get(appname, key string) (string, bool) {

	cache, ok := c.storage[appname]
	if !ok {
		return "", ok
	}

	su, ok := cache[key]
	if !ok {
		return "", ok
	}

	if time.Now().Unix() - su.CreatedAt >= su.TTL {
		delete(c.storage[appname], key)
		return "", false
	}

	return su.Value, ok
}

func (c *Cache) removeKey(appname, key string) bool {

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


func (c *Cache) removeApp(appname string) bool {

	_, ok := c.storage[appname]
	if !ok {
		return false
	}

	delete(c.storage, appname)
	return true
}

func (c *Cache) removeAll() {

	for app, _ := range c.storage {
		delete(c.storage, app)
	}
}

func (c *Cache) gc() {

	log.Println("cache::Starting GC")

	for app, cache := range c.storage {
		for k , su := range cache {
			if time.Now().Unix() - su.CreatedAt > su.TTL {
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

func (c *Cache) stats() *CacheStats {

	cs := &CacheStats{
		Node:  ME,
		Nodes: SIBLINGS_MANAGER.GetSiblings(),
		MemBytes: 0,
	}

	for app, cache := range c.storage {
		cs.Applications = append(cs.Applications, app)
		for k , su := range cache {
			cs.MemBytes += int64(len(k)) + int64(len(su.Value))

		}
	}

	return cs
}

type ExportUnit struct {
	AppName   string `json:"appname"`
	CreatedAt int64  `json:"created_at"`
	Key       string `json:"key"`
	Value     string `json:"val"`
	TTL       int64  `json:"ttl"`
}

func (c *Cache) contentExporter(ch chan *ExportUnit) {

	for app, cache := range c.storage {
		for k , su := range cache {
			eu := &ExportUnit{
				AppName: app,
				CreatedAt: su.CreatedAt,
				Key: k,
				Value: su.Value,
				TTL: su.TTL,
			}
			delete(c.storage[app], k)
			ch <-eu
			log.Printf("cache::export %+v ready to export\n", eu)
		}
	}
	close(ch)
}

func (c *Cache) importForeign(wop *writeOp) {
	su := StorageUnit {
		CreatedAt: wop.ct,
		Value: wop.val,
		TTL: wop.ttl,
	 }

	_, ok := c.storage[wop.app]
	if !ok {
		c.storage[wop.app] = map[string]StorageUnit{}
	}

	c.storage[wop.app][wop.key] = su
}



