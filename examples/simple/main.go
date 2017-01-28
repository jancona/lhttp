package main

import (
	"bytes"
	"fmt"
	"html"
	"log"
	"net/http"

	"github.com/jancona/lhttp"
)

type fooHandler struct{}

func (f fooHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "This is /foo")
}

func main() {
	lhttp.Handle("/foo", fooHandler{})

	lhttp.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Echo
		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "URL: %v\n", html.EscapeString(r.URL.String()))
		buf := new(bytes.Buffer)
		buf.ReadFrom(r.Body)
		fmt.Fprintf(w, "<p>Body: %v\n", html.EscapeString(buf.String()))
	})

	log.Fatal(lhttp.ListenAndServe("", nil))

}
