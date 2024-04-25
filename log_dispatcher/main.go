package main

import (
	"log"
	"net/http"
)

func main() {
	ds := newDispatchServer()
	s := &http.Server{
		Handler: ds,
		Addr:    ":8080",
	}
	log.Fatal(s.ListenAndServe())
}
