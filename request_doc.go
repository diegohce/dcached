package main

import (
	"fmt"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func cacheGetDoc(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	doc := `{
	"POST": {
		"description": "Returns value from dcached cluster",
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

	w.Header().Set("Allow", "POST,OPTIONS")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}

func cacheSetDoc(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	doc := `{
	"POST": {
		"description": "Stores value into dcached cluster",
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

	w.Header().Set("Allow", "POST,OPTIONS")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}

func cacheRemoveDoc(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	doc := ""
	cacheBlock := ps.ByName("cache_block")

	if cacheBlock == "application" {

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

	} else if cacheBlock == "key" {
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

	} else if cacheBlock == "all" {
		doc = `{
	"POST": {
		"description": "Wipes dcached cluster (all nodes)",
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
		e := newException("ResourceNotFoundException", "Resource not found")
		e.write(w)
		return
	}

	w.Header().Set("Allow", "POST,OPTIONS")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}

func cacheImportDoc(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	doc := `{
	"POST": {
		"description": "Reserved",
		"parameters": {}
	}
}`

	w.Header().Set("Allow", "POST,OPTIONS")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}

func cacheStatsHandlerDoc(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {

	doc := ""
	statsType := ps.ByName("stats_type")

	if statsType == "local" {

		doc = `{
	"GET": {
		"description": "Returns statistic from the requested node",
		"parameters": {}
	}
}`

	} else if statsType == "all" {
		doc = `{
	"GET": {
		"description": "Returns statistics from the whole cluster",
		"parameters": {}
	}
}`

	} else {
		e := newException("ResourceNotFoundException", "Resource not found")
		e.write(w)
		return
	}

	w.Header().Set("Allow", "GET,OPTIONS")
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s", doc)

}
