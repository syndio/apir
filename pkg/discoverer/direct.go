package discoverer

// Direct implements the Discoverer interface and just returns the given URL directly.
type Direct struct {
	url string
}

// NewDirect initializes a new Direct Discoverer with the given base URL.
func NewDirect(url string) *Direct {
	return &Direct{url: url}
}

// URL implements the Discoverer.URL method returning the URL for this Discoverer.
func (d *Direct) URL() string {
	return d.url
}
