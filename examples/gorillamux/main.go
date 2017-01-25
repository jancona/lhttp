package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/jancona/lambdaproxy/lhttp"
)

func articlesCategoryHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Category: %v\n", vars["category"])
}

func main() {
	r := mux.NewRouter()
	r.HandleFunc("/articles/{category}/", articlesCategoryHandler)

	lhttp.Handle("/", r)
	log.Fatal(lhttp.ListenAndServe("", nil))

}
