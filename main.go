package main

import (
	"io"
	"log"
	"os"
	//"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
)

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", IndexEndpoint).Methods("GET")

	log.Fatal(http.ListenAndServe(
		":8080",
		handlers.LoggingHandler(os.Stdout, router),
	))
}

func IndexEndpoint(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ololo")
}
