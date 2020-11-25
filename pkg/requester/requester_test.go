package requester_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"
	"time"

	"github.com/kpurdon/apir/pkg/discoverer"
	"github.com/kpurdon/apir/pkg/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	csvServer      *httptest.Server
	jsonServer     *httptest.Server
	timeoutServer  *httptest.Server
	failOnceServer *httptest.Server
	failOnceMap    = make(map[string]bool)
)

func TestMain(m *testing.M) {
	csvServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path.Join("files", "test.csv"))
	}))
	defer csvServer.Close()

	jsonServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"color":"red"}`)); err != nil {
			panic(err)
		}
	}))
	defer jsonServer.Close()

	timeoutServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer timeoutServer.Close()

	failOnceServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var shouldFail bool
		if _, ok := failOnceMap[r.URL.Path]; !ok {
			shouldFail = true
			failOnceMap[r.URL.Path] = true
		}

		if shouldFail {
			w.WriteHeader(http.StatusBadGateway)
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"color":"red"}`)); err != nil {
			panic(err)
		}
	}))
	defer failOnceServer.Close()

	os.Exit(m.Run()) //nolint:gocritic
}

func TestContentTypeString(t *testing.T) {
	t.Skip("TODO")
}

func TestNewClient(t *testing.T) {
	t.Skip("TODO")
}

func TestClientMustAddAPI(t *testing.T) {
	t.Skip("TODO")
}

func TestClientNewRequest(t *testing.T) {
	t.Skip("TODO")
}

func TestClientExecute_CSV(t *testing.T) {
	client := requester.NewClient("test")
	client.MustAddAPI("testcsv", discoverer.NewDirect(csvServer.URL),
		requester.WithContentType(requester.TextCSV))

	req, err := client.NewRequest(context.TODO(), "testcsv", http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.NotNil(t, req)

	var data bytes.Buffer
	ok, err := client.Execute(req, &data, nil)
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, "id,color\n1,red\n2,blue\n", data.String())
}

func TestClientExecute_JSON(t *testing.T) {
	client := requester.NewClient("test")
	client.MustAddAPI("testjson", discoverer.NewDirect(jsonServer.URL))

	req, err := client.NewRequest(context.TODO(), "testjson", http.MethodGet, "/", nil)
	require.NoError(t, err)
	assert.NotNil(t, req)

	var data struct {
		Color string `json:"color"`
	}
	ok, err := client.Execute(req, &data, nil)
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, "red", data.Color)
}

func TestClientExecute_Retry(t *testing.T) {
	client := requester.NewClient("test", requester.WithRetry())
	client.MustAddAPI("testjson-retry", discoverer.NewDirect(failOnceServer.URL),
		requester.WithContentType(requester.ApplicationJSON))

	req, err := client.NewRequest(context.TODO(), "testjson-retry", http.MethodGet,
		fmt.Sprintf("/%s", t.Name()), nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	var data struct {
		Color string `json:"color"`
	}
	ok, err := client.Execute(req, &data, nil)
	assert.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, "red", data.Color)
	assert.True(t, failOnceMap[req.URL.Path], "did not fail+retry")
}

func TestClientExecute_Timeout(t *testing.T) {
	client := requester.NewClient("test", requester.WithTimeout(100*time.Millisecond))
	client.MustAddAPI("timeout", discoverer.NewDirect(timeoutServer.URL),
		requester.WithContentType(requester.ApplicationJSON))

	req, err := client.NewRequest(context.TODO(), "timeout", http.MethodGet,
		fmt.Sprintf("/%s", t.Name()), nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	ok, err := client.Execute(req, nil, nil)
	assert.Error(t, err)
	assert.False(t, ok)
}

func TestRequestURL(t *testing.T) {
	t.Skip("TODO")
}
