package main

import (
	"log"
	"net/http"
)

func main() {
	fileserver := http.FileServer(http.Dir("./"))

	mux := http.NewServeMux()
	mux.Handle("/", fileserver)

	srv := &http.Server{
		Addr:    ":8000",
		Handler: mux,
	}

	log.Println("Serving on http://0.0.0.0|[::1]:8080")
	log.Fatal(srv.ListenAndServe())
}
