package requester

import (
	"context"
	"io"
)

// ensure MockClient implements Requester at compile time.
var _ Requester = &MockClient{}

// MockClient implements the Requester interface allowing for complete control over the returned values.
type MockClient struct { //nolint:maligned
	AddAPIFn       func(apiName string, discoverer Discoverer, options ...APIOption) error
	AddAPIFnCalled bool

	NewRequestFn       func(ctx context.Context, apiName, method, url string, body io.Reader, options ...RequestOption) (*Request, error)
	NewRequestFnCalled bool

	ExecuteFn       func(req *Request, successData, errorData interface{}) (bool, error)
	ExecuteFnCalled bool
}

// AddAPI implements the Requester.MustAddAPI method.
func (m *MockClient) AddAPI(apiName string, discoverer Discoverer, options ...APIOption) error {
	m.AddAPIFnCalled = true
	return m.AddAPIFn(apiName, discoverer, options...)
}

// NewRequest implements the Requester.NewRequest method.
func (m *MockClient) NewRequest(ctx context.Context, apiName, method, url string, body io.Reader, options ...RequestOption) (*Request, error) {
	m.NewRequestFnCalled = true
	return m.NewRequestFn(ctx, apiName, method, url, body, options...)
}

// Execute implements the Requester.Execute method.
func (m *MockClient) Execute(req *Request, successData, errorData interface{}) (bool, error) {
	m.ExecuteFnCalled = true
	return m.ExecuteFn(req, successData, errorData)
}
