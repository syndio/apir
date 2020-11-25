// Package requester provides and interface and implementation for creating and executing requests to APIs.
package requester

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

// ContentType is an http Content-Type value.
type ContentType string

// String implements the Stringer interface returning the Contenty-Type string.
func (c ContentType) String() string {
	return string(c)
}

var (
	// ApplicationJSON is the application/json Content-Type.
	ApplicationJSON ContentType = "application/json"

	// TextCSV is the text/csv Content-Type.
	TextCSV ContentType = "text/csv"
)

// Discoverer defines an interface that allows for dynamic was of discovering a URL for a request.
type Discoverer interface {
	URL() string
}

// Requester defines an interface for creating and executing requests to an API.
type Requester interface {
	MustAddAPI(apiName string, discoverer Discoverer, options ...APIOption)
	NewRequest(ctx context.Context, apiName, method, url string, body io.Reader, options ...RequestOption) (*Request, error)
	Execute(req *Request, successData, errorData interface{}) (bool, error)
}

// API defines an API and is embedded in a Client via MustAddAPI.
type API struct {
	Discoverer
	contentType ContentType
}

// APIOption defines configuration options for an API.
type APIOption func(*API)

// WithContentType sets the ContentType for an API. If not specified the default ApplicationJSON is used.
func WithContentType(ct ContentType) APIOption {
	return func(api *API) {
		api.contentType = ct
	}
}

// ensure Client implements Requester at compile time.
var _ Requester = &Client{}

// Client implements the Requester interface.
type Client struct {
	name   string
	client *http.Client
	apis   map[string]*API
}

// ClientOption defines configuration options for a Client.
type ClientOption func(*Client)

// WithClient sets the underlying *http.Client for a Client. Replaces any existing *http.Client.
func WithClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.client = hc
	}
}

// WithRetry sets the underlying *http.Client with one configured for automated retry. Replaces any existing *http.Client.
func WithRetry() ClientOption {
	return func(c *Client) {
		rc := retryablehttp.NewClient()
		c.client = rc.StandardClient()
	}
}

// WithTimeout sets the *http.Client.Timeout to the provided value. Be sure to call this after configuring any *http.Client (e.g. WithClient, WithRetry, ...).
func WithTimeout(t time.Duration) ClientOption {
	return func(c *Client) {
		c.client.Timeout = t
	}
}

// NewClient creates a new Client with sane defaults and applies any given ClientOption methods.
func NewClient(name string, options ...ClientOption) *Client {
	c := &Client{
		name:   name,
		client: http.DefaultClient,
		apis:   make(map[string]*API),
	}
	for _, option := range options {
		option(c)
	}
	return c
}

// MustAddAPI adds an API with the given name and Discover to the Client applying any given APIOption methods.
func (c *Client) MustAddAPI(name string, discoverer Discoverer, options ...APIOption) {
	if _, ok := c.apis[name]; ok {
		panic(fmt.Sprintf("api %q already initialized", name))
	}

	api := &API{Discoverer: discoverer}
	for _, option := range options {
		option(api)
	}

	// if the WithContentType option was no applied ensure we set the default value of ApplicationJSON
	if api.contentType == "" {
		api.contentType = ApplicationJSON
	}

	c.apis[name] = api
}

// Request defines a http request to be made to an API.
type Request struct {
	api       *API
	userAgent string
	*http.Request
}

// RequestOption defines configuration options for a Request.
type RequestOption func(*Request)

// WithUserAgent sets the user agent to be used on the Request.
func WithUserAgent(ua string) RequestOption {
	return func(r *Request) {
		r.userAgent = ua
	}
}

// NewRequest creates a new Request for the given inputs applying any given RequestOption methods.
func (c *Client) NewRequest(ctx context.Context, apiName, method, url string, body io.Reader, options ...RequestOption) (*Request, error) {
	api, ok := c.apis[apiName]
	if !ok {
		return nil, fmt.Errorf("api %q not initialized", apiName)
	}

	u := fmt.Sprintf("%s/%s", strings.TrimRight(api.URL(), "/"), strings.TrimLeft(url, "/"))
	req, err := http.NewRequest(method, u, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// set the default requests content type based on the api content type
	if body != nil {
		req.Header.Set("Content-Type", api.contentType.String())
	}

	// set the default user agent (can be changed w/ the WithUserAgent option)
	req.Header.Set("User-Agent", fmt.Sprintf("kpurdon/apir (for %s)", c.name))

	r := &Request{api: api, Request: req}
	for _, option := range options {
		option(r)
	}

	return r, nil
}

// Execute makes the given Request optionally decoding the response into given successData and/or errorData. The bool value returned indicates if the request was made successfully or not regardless of the response.
func (c *Client) Execute(req *Request, successData, errorData interface{}) (bool, error) {
	resp, err := c.client.Do(req.Request)
	if err != nil {
		return false, fmt.Errorf("error making request: %w", err)
	}
	defer func() {
		err = resp.Body.Close()
		if err != nil {
			fmt.Printf("error closing response body: %+v", err)
		}
	}()

	var ok bool
	switch req.api.contentType {
	case ApplicationJSON:
		ok, err = decodeJSON(resp, successData, errorData)
	case TextCSV:
		// TODO: decodeFile does not currently support errorData
		ok, err = decodeFile(resp, successData)
	default:
		return false, fmt.Errorf("content type %q not implemented", req.api.contentType)
	}

	return ok, err
}

func decodeJSON(resp *http.Response, successData, errorData interface{}) (bool, error) {
	if resp.StatusCode >= http.StatusBadRequest {
		if errorData != nil {
			if err := json.NewDecoder(resp.Body).Decode(&errorData); err != nil {
				return false, fmt.Errorf("decoding errorData: %w", err)
			}
			return false, nil
		}
		// TODO: better error situation here
		return false, fmt.Errorf("%d:%s", resp.StatusCode, resp.Body)
	}
	if successData != nil {
		if err := json.NewDecoder(resp.Body).Decode(&successData); err != nil {
			return true, fmt.Errorf("decoding successData: %w", err)
		}
	}
	return true, nil
}

func decodeFile(resp *http.Response, successData interface{}) (bool, error) {
	if resp.StatusCode >= http.StatusBadRequest {
		return false, fmt.Errorf("%d:%s", resp.StatusCode, resp.Body)
	}
	if successData != nil {
		w, ok := successData.(io.Writer)
		if !ok {
			return false, errors.New("successData must be an io.Writer for file decoding")
		}
		if _, err := io.Copy(w, resp.Body); err != nil {
			return true, fmt.Errorf("copying resp.Body to successData: %w", err)
		}
	}
	return true, nil
}
