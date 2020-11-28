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
	retryMap = make(map[string]bool)
	ts       *httptest.Server
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Query().Get("test") {
	case "file-csv":
		http.ServeFile(w, r, path.Join("files", "test.csv"))
		return
	case "json":
		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"color":"red"}`)); err != nil {
			panic(err)
		}
		return
	case "json-error":
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		if _, err := w.Write([]byte(`{"message":"bad"}`)); err != nil {
			panic(err)
		}
		return
	case "timeout":
		time.Sleep(1 * time.Second)
		w.WriteHeader(http.StatusOK)
		return
	case "retry":
		var shouldFail bool
		if _, ok := retryMap[r.URL.Path]; !ok {
			shouldFail = true
			retryMap[r.URL.Path] = true
		}

		if shouldFail {
			w.WriteHeader(http.StatusBadGateway)
		}

		w.Header().Set("Content-Type", "application/json")
		if _, err := w.Write([]byte(`{"color":"red"}`)); err != nil {
			panic(err)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func TestMain(m *testing.M) {
	ts = httptest.NewServer(http.HandlerFunc(testHandler))
	code := m.Run()
	ts.Close()
	os.Exit(code)
}

func TestContentTypeString(t *testing.T) {
	assert.Equal(t, "application/json", requester.ApplicationJSON.String())
}

func TestNewClient(t *testing.T) {
	assert.NotNil(t, requester.NewClient("test"))
}

func TestClientMustAddAPI(t *testing.T) {
	client := requester.NewClient("test")
	require.NotNil(t, client)
	client.MustAddAPI("test", discoverer.NewDirect(ts.URL))
	assert.Panics(t, func() {
		client.MustAddAPI("test", discoverer.NewDirect(ts.URL))
	}, "cannot add the same api twice")
}

func TestClientNewRequest(t *testing.T) {
	client := requester.NewClient("test")
	require.NotNil(t, client)
	client.MustAddAPI("test", discoverer.NewDirect(ts.URL))

	req, err := client.NewRequest(context.TODO(), "not-test", http.MethodGet, "", nil)
	assert.Nil(t, req)
	assert.Error(t, err)

	req, err = client.NewRequest(context.TODO(), "test", http.MethodGet, "", nil)
	assert.NotNil(t, req)
	assert.NoError(t, err)
}

func TestClientExecute_CSV(t *testing.T) {
	client := requester.NewClient("test")
	client.MustAddAPI("file-csv", discoverer.NewDirect(ts.URL),
		requester.WithContentType(requester.TextCSV))

	req, err := client.NewRequest(context.TODO(), "file-csv", http.MethodGet, "/?test=file-csv", nil)
	require.NoError(t, err)
	assert.NotNil(t, req)

	t.Run("success", func(t *testing.T) {
		var data bytes.Buffer
		ok, err := client.Execute(req, &data, nil)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "id,color\n1,red\n2,blue\n", data.String())
	})

	t.Run("not a buffer", func(t *testing.T) {
		var data struct {
			Foo string `json:"foo"`
		}
		ok, err := client.Execute(req, &data, nil)
		require.Error(t, err)
		assert.False(t, ok)
	})
}

func TestClientExecute_JSON(t *testing.T) {
	client := requester.NewClient("test")
	client.MustAddAPI("json", discoverer.NewDirect(ts.URL))

	req, err := client.NewRequest(context.TODO(), "json", http.MethodGet, "/?test=json", nil)
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

func TestClientExecute_JSONError(t *testing.T) {
	client := requester.NewClient("test")
	client.MustAddAPI("json-error", discoverer.NewDirect(ts.URL))

	req, err := client.NewRequest(context.TODO(), "json-error", http.MethodGet, "/?test=json-error", nil)
	require.NoError(t, err)
	assert.NotNil(t, req)

	var errorData struct {
		Message string `json:"message"`
	}
	ok, err := client.Execute(req, nil, &errorData)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, "bad", errorData.Message)
}

func TestClientExecute_NoErrorData(t *testing.T) {
	client := requester.NewClient("test")
	client.MustAddAPI("json-error", discoverer.NewDirect(ts.URL))

	req, err := client.NewRequest(context.TODO(), "json-error", http.MethodGet, "/?test=json-error", nil)
	require.NoError(t, err)
	assert.NotNil(t, req)

	var data struct {
		Color string `json:"color"`
	}
	ok, err := client.Execute(req, &data, nil)
	require.Error(t, err)
	assert.False(t, ok)
	// assert.Equal(t, "foo", errorData.Message)
}

func TestClientExecute_Retry(t *testing.T) {
	client := requester.NewClient(t.Name(), requester.WithRetry())
	client.MustAddAPI("retry", discoverer.NewDirect(ts.URL),
		requester.WithContentType(requester.ApplicationJSON))

	req, err := client.NewRequest(context.TODO(), "retry", http.MethodGet,
		fmt.Sprintf("/%s?test=retry", t.Name()), nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	var data struct {
		Color string `json:"color"`
	}
	ok, err := client.Execute(req, &data, nil)
	assert.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, "red", data.Color)
	assert.True(t, retryMap[req.URL.Path], "no retry recorded")
}

func TestClientExecute_Timeout(t *testing.T) {
	client := requester.NewClient("test", requester.WithTimeout(100*time.Millisecond))
	client.MustAddAPI("timeout", discoverer.NewDirect(ts.URL),
		requester.WithContentType(requester.ApplicationJSON))

	req, err := client.NewRequest(context.TODO(), "timeout", http.MethodGet,
		fmt.Sprintf("/%s?test=timeout", t.Name()), nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	ok, err := client.Execute(req, nil, nil)
	assert.Error(t, err)
	assert.False(t, ok)
}
