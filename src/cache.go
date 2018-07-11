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
	done chan bool
}


type removeOp struct {
	app   string
	key   string
	found bool
	done  chan bool
}

type StorageUnit struct {
	CreatedAt int64
	Value     string
	TTL       int64
}


type AppCache map[string]map[string]StorageUnit

type Cache struct {
	storage   AppCache
	Reads     chan *readOp
	Writes    chan *writeOp
	RemoveKey chan *removeOp
	RemoveApp chan *removeOp
	GcTimer   *time.Ticker
}


var (
	CACHE *Cache
)


func init() {
	CACHE = NewCache()
}


func NewCache() *Cache {

	c := &Cache{storage: AppCache{},
		Reads: make(chan *readOp),
        Writes: make(chan *writeOp),
		RemoveKey: make(chan *removeOp),
		RemoveApp: make(chan *removeOp),
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

func (c *Cache) gc() {

	log.Println("cache::Starting GC")

	for app, cache := range c.storage {
		for k , su := range cache {
			if time.Now().Unix() - su.CreatedAt > su.TTL {
				delete(c.storage[app], k)
				log.Printf("cache::GC %s->%s has been killed", app, k)
			}
		}
	}
	log.Println("cache::GC finished")
}


