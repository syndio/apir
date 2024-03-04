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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/syndio/apir/pkg/discoverer"
	"github.com/syndio/apir/pkg/requester"
)

var (
	retryMap = make(map[string]bool)
	ts       *httptest.Server
)

func testHandler(w http.ResponseWriter, r *http.Request) { //nolint:cyclop
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
	t.Parallel()
	assert.Equal(t, "application/json", requester.ApplicationJSON.String())
}

func TestNewClient(t *testing.T) {
	t.Parallel()
	assert.NotNil(t, requester.NewClient("test"))
}

func TestClientAddAPI(t *testing.T) {
	t.Parallel()
	client := requester.NewClient("test")
	require.NotNil(t, client)
	require.NoError(t, client.AddAPI("test", discoverer.NewDirect(ts.URL)))
	require.Error(t, client.AddAPI("test", discoverer.NewDirect(ts.URL)), "cannot add the same api twice")
	require.Error(
		t,
		client.AddAPI(
			"test-bad-url",
			discoverer.NewDirect("http://user:}{@foo.com")),
		"discoverer must return a valid url")
}

func TestClientNewRequest(t *testing.T) {
	t.Parallel()
	client := requester.NewClient("test")
	require.NotNil(t, client)
	require.NoError(t, client.AddAPI("test", discoverer.NewDirect(ts.URL)))

	req, err := client.NewRequest(context.TODO(), "not-test", http.MethodGet, "", nil)
	assert.Nil(t, req)
	require.Error(t, err)

	req, err = client.NewRequest(context.TODO(), "test", http.MethodGet, "", nil)
	assert.NotNil(t, req)
	require.NoError(t, err)
}

func TestClientExecute_CSV(t *testing.T) {
	t.Parallel()
	client := requester.NewClient("test")
	require.NoError(t, client.AddAPI("file-csv", discoverer.NewDirect(ts.URL),
		requester.WithContentType(requester.TextCSV)))

	req, err := client.NewRequest(context.TODO(), "file-csv", http.MethodGet, "/?test=file-csv", nil)
	require.NoError(t, err)
	assert.NotNil(t, req)

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		var data bytes.Buffer
		ok, err := client.Execute(req, &data, nil)
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, "id,color\n1,red\n2,blue\n", data.String())
	})

	t.Run("not a buffer", func(t *testing.T) {
		t.Parallel()
		var data struct {
			Foo string `json:"foo"`
		}
		ok, err := client.Execute(req, &data, nil)
		require.Error(t, err)
		assert.False(t, ok)
	})
}

func TestClientExecute_JSON(t *testing.T) {
	t.Parallel()
	client := requester.NewClient("test")
	require.NoError(t, client.AddAPI("json", discoverer.NewDirect(ts.URL)))

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
	t.Parallel()
	client := requester.NewClient("test")
	require.NoError(t, client.AddAPI("json-error", discoverer.NewDirect(ts.URL)))

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
	t.Parallel()
	client := requester.NewClient("test")
	require.NoError(t, client.AddAPI("json-error", discoverer.NewDirect(ts.URL)))

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
	t.Parallel()
	client := requester.NewClient(t.Name(), requester.WithRetry())
	require.NoError(t, client.AddAPI("retry", discoverer.NewDirect(ts.URL),
		requester.WithContentType(requester.ApplicationJSON)))

	req, err := client.NewRequest(context.TODO(), "retry", http.MethodGet,
		fmt.Sprintf("/%s?test=retry", t.Name()), nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	var data struct {
		Color string `json:"color"`
	}
	ok, err := client.Execute(req, &data, nil)
	require.NoError(t, err)
	assert.True(t, ok)

	assert.Equal(t, "red", data.Color)
	assert.True(t, retryMap[req.URL.Path], "no retry recorded")
}

func TestClientExecute_Timeout(t *testing.T) {
	t.Parallel()
	client := requester.NewClient("test", requester.WithTimeout(100*time.Millisecond))
	require.NoError(t, client.AddAPI("timeout", discoverer.NewDirect(ts.URL),
		requester.WithContentType(requester.ApplicationJSON)))

	req, err := client.NewRequest(context.TODO(), "timeout", http.MethodGet,
		fmt.Sprintf("/%s?test=timeout", t.Name()), nil)
	require.NoError(t, err)
	require.NotNil(t, req)

	ok, err := client.Execute(req, nil, nil)
	require.Error(t, err)
	assert.False(t, ok)
}
