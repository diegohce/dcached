package main

import (
	"fmt"
	"net/http"
    "github.com/julienschmidt/httprouter"
)

func CacheGetDoc(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	doc := `{
	"POST": {
		"description": "Retrieves value from dcached cluster",
		"parameters": {
			"appname": {
				"type": "string",
				"description": "Application name requesting value",
				"required": true
			},
			"key": {
				"type": "string",
				"description": "The key to retrieve",
				"required": true
			}
		}
	}
}`

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}


func CacheSetDoc(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	doc := `{
	"POST": {
		"description": "Stores value to dcached cluster",
		"parameters": {
			"appname": {
				"type": "string",
				"description": "Application name storing value",
				"required": true
			},
			"key": {
				"type": "string",
				"description": "The key to set",
				"required": true
			},
			"value": {
				"type": "string",
				"description": "The value you want to cache",
				"required": true
			},
			"ttl": {
				"type": "integer",
				"description": "Seconds since creation to invalidate this value",
				"required": true
			}
		}
	}
}`

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}


func CacheRemoveDoc(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	doc := ""
	cache_block := ps.ByName("cache_block")

	if cache_block == "application" {

		doc = `{
	"POST": {
		"description": "Removes application workspace from dcached cluster",
		"parameters": {
			"appname": {
				"type": "string",
				"description": "Application name to remove",
				"required": true
			}
		}
	}
}`

	} else if cache_block == "key" {
		doc = `{
	"POST": {
		"description": "Removes value from dcached cluster",
		"parameters": {
			"appname": {
				"type": "string",
				"description": "Application name to remove key from",
				"required": true
			},
			"key": {
				"type": "string",
				"description": "The key to remove",
				"required": true
			}
		}
	}
}`

	} else if cache_block == "all" {
		doc = `{
	"POST": {
		"description": "Erase dcached cluster (all nodes)",
		"parameters": {
			"appname": {
				"type": "string",
				"description": "Application name, must be '*'",
				"required": true
			}
		}
	}
}`
	} else {
		e := NewException("ResourceNotFoundException", "Resource not found")
		e.Write(w)
		return
	}


	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}

func CacheImportDoc(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	doc := `{
	"POST": {
		"description": "Reserved",
		"parameters": {}
	}
}`

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}



func CacheStatsHandlerDoc(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	doc := ""
	stats_type := ps.ByName("stats_type")

	if stats_type == "local" {

		doc = `{
	"GET": {
		"description": "Returns statistic from the requested node",
		"parameters": {}
	}
}`

	} else if stats_type == "all" {
		doc = `{
	"GET": {
		"description": "Returs statistics from the whole cluster",
		"parameters": {}
	}
}`

	} else {
		e := NewException("ResourceNotFoundException", "Resource not found")
		e.Write(w)
		return
	}


	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}



