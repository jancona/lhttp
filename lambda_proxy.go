// Copyright 2017 James P. Ancona
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package lhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
)

// payload is the request payload as passed from index.js
type payload struct {
	// Lambda proxy request
	Request *lambdaRequest `json:"event"`
	// default Context object
	Context *LambdaContext `json:"context"`
}

// LambdaContext is the AWS Lambda request context
type LambdaContext struct {
	AwsRequestID             string `json:"awsRequestId"`
	FunctionName             string `json:"functionName"`
	FunctionVersion          string `json:"functionVersion"`
	InvokeID                 string `json:"invokeid"`
	IsDefaultFunctionVersion bool   `json:"isDefaultFunctionVersion"`
	LogGroupName             string `json:"logGroupName"`
	LogStreamName            string `json:"logStreamName"`
	MemoryLimitInMB          string `json:"memoryLimitInMB"`
}

// lambdaRequest is the AWS proxy integration request
type lambdaRequest struct {
	Resource              string            `json:"resource"`
	Path                  string            `json:"path"`
	HTTPMethod            string            `json:"httpMethod"`
	Headers               map[string]string `json:"headers"`
	QueryStringParameters map[string]string `json:"queryStringParameters"`
	PathParameters        map[string]string `json:"pathParameters"`
	StageVariables        map[string]string `json:"stageVariables"`
	RequestContext        requestContext    `json:"requestContext"`
	Body                  string            `json:"body"`
	IsBase64Encoded       bool              `json:"isBase64Encoded"`
}

func (lr lambdaRequest) url() string {
	// TODO: It's not clear what the right thing to do here is.
	// The commented code probably doesn't work for custom domains, plus
	// the URL mapping in handlers would have to include the stage
	// path segment (which can't be known when the mapping is set up).
	// So for the time being, make the path in the request URL be the
	// part after the stage. This will break redirects when a stage
	// segment is present.
	url :=
		// lr.Headers["X-Forwarded-Proto"] + "://" +
		// lr.Headers["Host"] +
		// ":" + lr.Headers["X-Forwarded-Port"] +
		// "/" + lr.RequestContext.Stage +
		lr.Path
	join := "?"
	for k, v := range lr.QueryStringParameters {
		url += join + k + "=" + v
		join = "&"
	}
	return url
}

type handler func(*LambdaContext, *lambdaRequest) (interface{}, error)

type requestContext struct {
	AccountID    string            `json:"accountId"`
	ResourceID   string            `json:"resourceId"`
	Stage        string            `json:"stage"`
	RequestID    string            `json:"requestId"`
	Identity     map[string]string `json:"identity"`
	ResourcePath string            `json:"resourcePath"`
	HTTPMethod   string            `json:"httpMethod"`
	APIID        string            `json:"apiId"`
}

type lambdaResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body"`
}

func newLambdaResponse(r *response) *lambdaResponse {
	lr := &lambdaResponse{
		r.status,
		map[string]string{},
		r.body,
	}
	for k, v := range r.Header() {
		lr.Headers[k] = v[0]
	}
	return lr
}

func newErrorResponse(err error) *lambdaResponse {
	e := err.Error()
	return &lambdaResponse{
		StatusCode: 500,
		Body:       e,
	}
}

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation.
type contextKey struct {
	name string
}

func (k *contextKey) String() string { return "lhttp context value " + k.name }

// LambdaContextKey is a context key. It can be used by handlers to access the
// AWS Lambda Context from the Request object. The associated value will be of
// type LambdaContext
var LambdaContextKey = contextKey{"lambda-context"}

// DefaultServeMux is the default ServeMux used by Serve.
var DefaultServeMux = &defaultServeMux

var defaultServeMux http.ServeMux

// Handle registers the handler for the given pattern
// in the DefaultServeMux.
// The documentation for http.ServeMux explains how patterns are matched.
func Handle(pattern string, handler http.Handler) { DefaultServeMux.Handle(pattern, handler) }

// HandleFunc registers the handler function for the given pattern
// in the DefaultServeMux.
// The documentation for http.ServeMux explains how patterns are matched.
func HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}

type response struct {
	status int
	body   string
	header http.Header
}

func (r *response) Header() http.Header {
	return r.header
}
func (r *response) WriteHeader(code int) {
	r.status = code
}
func (r *response) Write(data []byte) (n int, err error) {
	r.body += string(data)
	return len(data), nil
}

// LambdaServer is an analog of http.Server to handle Lambda requests
type LambdaServer struct {
	Handler http.Handler // handler to invoke, http.DefaultServeMux if nil
}

// ListenAndServe listens for incoming requests and dispatches them
// to the configured handler
func (srv *LambdaServer) ListenAndServe() error {
	return srv.listenAndServe(os.Stdin, os.Stdout)
}

// listenAndServe is factored to make it unit test-able
func (srv *LambdaServer) listenAndServe(in io.Reader, out io.Writer) error {
	runStream(func(context *LambdaContext, lr *lambdaRequest) (interface{}, error) {
		handler := srv.Handler
		if handler == nil {
			handler = DefaultServeMux
		}
		rw := new(response)
		rw.status = http.StatusOK
		rw.header = http.Header{}
		req := newHTTPRequest(lr, context)
		handler.ServeHTTP(rw, req)
		return newLambdaResponse(rw), nil
	}, in, out)
	return nil
}

func newHTTPRequest(lr *lambdaRequest, ctx *LambdaContext) *http.Request {
	hr, err := http.NewRequest(lr.HTTPMethod, lr.url(), strings.NewReader(lr.Body))
	if err != nil {
		log.Print("Error creating http.Request: ", err.Error())
	}
	hr.WithContext(context.WithValue(hr.Context(), &LambdaContextKey, ctx))
	return hr
}

func runStream(h handler, in io.Reader, out io.Writer) {
	stdin := json.NewDecoder(in)
	stdout := json.NewEncoder(out)

	for {
		if err := func() (err error) {
			defer func() {
				if e := recover(); e != nil {
					log.Printf("panic: %v, %s\n", e, debug.Stack())
					err = fmt.Errorf("panic: %v, %s", e, debug.Stack())
				}
			}()
			var p payload
			if er := stdin.Decode(&p); er != nil {
				return er
			}
			data, err := h(p.Context, p.Request)
			if err != nil {
				return err
			}
			return stdout.Encode(data)
		}(); err != nil {
			if encErr := stdout.Encode(newErrorResponse(err)); encErr != nil {
				// bad times
				log.Println("Failed to encode err response!", encErr.Error(), debug.Stack())
			}
		}
	}
}

// ListenAndServe listens for incoming requests and dispatches them
// to the configured handler
func ListenAndServe(addr string, handler http.Handler) error {
	server := &LambdaServer{}
	return server.ListenAndServe()
}
