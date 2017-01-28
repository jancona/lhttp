# lhttp: a Go Lambda Proxy

With lhttp you can deploy idiomatic Go HTTP server code as an AWS Lambda function
using AWS Api Gateway.

For example, this is a basic Go server from the http package Godoc:
```
http.Handle("/foo", fooHandler)

http.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
})

log.Fatal(http.ListenAndServe(":8080", nil))
```

This is the equivalent implementation using lhttp:
```
lhttp.Handle("/foo", fooHandler)

lhttp.HandleFunc("/bar", func(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
})

log.Fatal(lhttp.ListenAndServe("", nil))
```
See the [examples](examples/README.md) for more details.
