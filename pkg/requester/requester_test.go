package requester_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/kpurdon/apir/pkg/discoverer"
	"github.com/kpurdon/apir/pkg/requester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	csvServer  *httptest.Server
	jsonServer *httptest.Server
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
	t.Run("csv", func(t *testing.T) {
		client := requester.NewClient("test")
		client.MustAddAPI("testcsv", discoverer.NewDirect(csvServer.URL),
			requester.SetContentType(requester.TextCSV))

		req, err := client.NewRequest(context.TODO(), "testcsv", http.MethodGet, "/", nil)
		require.NoError(t, err)
		assert.NotNil(t, req)

		var data bytes.Buffer
		ok, err := client.Execute(req, &data, nil)
		require.NoError(t, err)
		assert.True(t, ok)

		assert.Equal(t, "id,color\n1,red\n2,blue\n", data.String())
	})
	t.Run("json", func(t *testing.T) {
		client := requester.NewClient("test")
		client.MustAddAPI("testjson", discoverer.NewDirect(jsonServer.URL),
			requester.SetContentType(requester.ApplicationJSON))

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
	})
}
