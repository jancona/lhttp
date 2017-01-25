package lhttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

// Sample Lambda Proxy request
var awsRequest = `{"event":{
  "body": "%s",
  "resource": "/{proxy+}",
  "requestContext": {
    "resourceId": "123456",
    "apiId": "1234567890",
    "resourcePath": "/{proxy+}",
    "httpMethod": "POST",
    "requestId": "c6af9ac6-7b61-11e6-9a41-93e8deadbeef",
    "accountId": "123456789012",
    "identity": {
      "apiKey": null,
      "userArn": null,
      "cognitoAuthenticationType": null,
      "caller": null,
      "userAgent": "Custom User Agent String",
      "user": null,
      "cognitoIdentityPoolId": null,
      "cognitoIdentityId": null,
      "cognitoAuthenticationProvider": null,
      "sourceIp": "127.0.0.1",
      "accountId": null
    },
    "stage": "prod"
  },
  "queryStringParameters": {
    "foo": "bar"
  },
  "headers": {
    "Via": "1.1 08f323deadbeefa7af34d5feb414ce27.cloudfront.net (CloudFront)",
    "Accept-Language": "en-US,en;q=0.8",
    "CloudFront-Is-Desktop-Viewer": "true",
    "CloudFront-Is-SmartTV-Viewer": "false",
    "CloudFront-Is-Mobile-Viewer": "false",
    "X-Forwarded-For": "127.0.0.1, 127.0.0.2",
    "CloudFront-Viewer-Country": "US",
    "Accept": "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
    "Upgrade-Insecure-Requests": "1",
    "X-Forwarded-Port": "443",
    "Host": "1234567890.execute-api.us-east-1.amazonaws.com",
    "X-Forwarded-Proto": "https",
    "X-Amz-Cf-Id": "cDehVQoZnx43VYQb9j2-nvCh-9z396Uhbp027Y2JvkCPNLmGJHqlaA==",
    "CloudFront-Is-Tablet-Viewer": "false",
    "Cache-Control": "max-age=0",
    "User-Agent": "Custom User Agent String",
    "CloudFront-Forwarded-Proto": "https",
    "Accept-Encoding": "gzip, deflate, sdch"
  },
  "pathParameters": {
    "proxy": "path/to/resource"
  },
  "httpMethod": "%s",
  "stageVariables": {
    "baz": "qux"
  },
  "path": "%s"
},"context": {}}`

type test struct {
	body   string
	method string
	path   string
	result lambdaResponse
}

var tests = []test{
	test{
		body:   "",
		method: "GET",
		path:   "/foo",
		result: lambdaResponse{
			StatusCode: 200,
			Body:       "foo",
		},
	},
	test{
		body:   "",
		method: "GET",
		path:   "/bar",
		result: lambdaResponse{
			StatusCode: 404,
			Body:       "404 page not found\n",
		},
	},
}

type fooHandler struct{}

func (f fooHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "foo")
}

func TestListenAndServe(t *testing.T) {
	requests := &bytes.Buffer{}
	for _, tst := range tests {
		fmt.Fprintf(requests, awsRequest, tst.body, tst.method, tst.path)
	}

	Handle("/foo", fooHandler{})
	server := new(LambdaServer)
	r, w := io.Pipe()
	go func() {
		server.listenAndServe(requests, w)
	}()
	dec := json.NewDecoder(r)
	for _, tst := range tests {

		var resp lambdaResponse
		if err := dec.Decode(&resp); err != nil {
			t.Error(err)
		}
		if resp.StatusCode != tst.result.StatusCode {
			t.Errorf("Expected StatusCode %v, got %v", tst.result.StatusCode, resp.StatusCode)
		}
		if resp.Body != tst.result.Body {
			t.Errorf("Expected Body '%v', got '%v'", tst.result.Body, resp.Body)
		}
	}
}
