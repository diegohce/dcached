package main



import (
	//"fmt"
	"time"
)

type StorageUnitNotFound error



type StorageUnit struct {
	CreatedAt int64
	Value     string
	TTL       int64
}


type AppCache map[string]map[string]StorageUnit

type Cache struct {
	storage AppCache
}


var (
	CACHE *Cache
)


func init() {
	CACHE = NewCache()
}


func NewCache() *Cache {

	c := &Cache{storage: AppCache{} }

	//start cache GC

	return c
}


func (c *Cache) Set(appname, key, value string, ttl int64) {

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

func (c *Cache) Get(appname, key string) (string, bool) {

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


