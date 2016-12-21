package main

import (
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/", indexEndpoint).Methods("GET")

	log.Fatal(http.ListenAndServe(
		":8080",
		handlers.LoggingHandler(os.Stdout, router),
	))
}

func indexEndpoint(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.gohtml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = t.Execute(w, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
