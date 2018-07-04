package main



import (
	//"fmt"
	//"log"
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


type StorageUnit struct {
	CreatedAt int64
	Value     string
	TTL       int64
}


type AppCache map[string]map[string]StorageUnit

type Cache struct {
	storage AppCache
	Reads  chan *readOp
	Writes chan *writeOp
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
            Writes: make(chan *writeOp) }

	go func(){
		for {
			select {
				case read := <-c.Reads:
				{
					//log.Println("Received readop")
					read.val, read.found = c.get(read.app, read.key)
					read.done <- true
				}
				case write := <-c.Writes:
				{
					//log.Println("Received writeop")
					c.set(write.app, write.key, write.val, write.ttl)
					write.done <- true
				}
			}
		}
	}()

	//start cache GC

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


