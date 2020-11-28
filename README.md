![go](https://github.com/kpurdon/apir/workflows/go/badge.svg)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/kpurdon/apir)](https://pkg.go.dev/github.com/kpurdon/apir)
[![Go Report Card](https://goreportcard.com/badge/github.com/kpurdon/apir)](https://goreportcard.com/report/github.com/kpurdon/apir)
[![codecov](https://codecov.io/gh/kpurdon/apir/branch/main/graph/badge.svg?token=XA8RTRD2SU)](https://codecov.io/gh/kpurdon/apir)

apir
-----

A simple package for making/consuming api[r] requests/responses.

## Installation

`go get github.com/kpurdon/apir`

## Getting Started


### Initialize a Client

```go
client := requester.NewClient("myservice")
client.MustAddAPI("otherservice", discoverer.NewDirect("http://foo.com/api"),
	requester.WithRetry(),
	requester.WithContentType(requester.ApplicationJSON),
)
```

### Create a Request

```go
req, err := client.NewRequest(ctx, "otherservice", http.MethodGet, "/", nil)
if err != nil {
    // an error occured trying to make the request
    // handle the error
}
```

### Execute the Request

```go
var (
   data struct{}
   errData struct{}
)
ok, err := client.Execute(req, &data, &errData)
if err != nil {
    // an error occured trying to executre the request
    // handle the error
    return
}
if !ok {
    // we made the request, but got a >= 400 status code
    // examine errData for error details
    return
}

// the request was made and returned a < 400 status code
// examine data for the response
```
