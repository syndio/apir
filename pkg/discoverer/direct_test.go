package discoverer_test

import (
	"testing"

	"github.com/kpurdon/apir/pkg/discoverer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDirect(t *testing.T) {
	t.Parallel()
	require.NotNil(t, discoverer.NewDirect("http://foo.bar/"))
}

func TestDirectURL(t *testing.T) {
	t.Parallel()
	u := "http://foo.bar/"
	d := discoverer.NewDirect(u)
	require.NotNil(t, d)
	assert.Equal(t, u, d.URL())
}
