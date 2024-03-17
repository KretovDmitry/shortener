package gzip

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommonCaseFilter_ShouldCompress(t *testing.T) {
	tests := []struct {
		name string
		req  *http.Request
		want bool
	}{
		{
			name: "Good request",
			req: &http.Request{
				Method: http.MethodPost,
				Header: map[string][]string{"Accept-Encoding": {"gzip"}},
			},
			want: true,
		},
		{
			name: "HEAD request",
			req: &http.Request{
				Method: http.MethodHead,
				Header: map[string][]string{"Accept-Encoding": {"gzip"}},
			},
			want: false,
		},
		{
			name: "OPTIONS request",
			req: &http.Request{
				Method: http.MethodOptions,
				Header: map[string][]string{"Accept-Encoding": {"gzip"}},
			},
			want: false,
		},
		{
			name: "HTTP2 upgrade request",
			req: &http.Request{
				Method: http.MethodPost,
				Header: map[string][]string{
					"Accept-Encoding": {"gzip"}, "Upgrade": {"http2"},
				},
			},
			want: false,
		},
		{
			name: "Not accepting gzip request",
			req:  &http.Request{Method: http.MethodPost},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCommonRequestFilter()
			assert.Equal(t, tt.want, c.ShouldCompress(tt.req))
		})
	}
}

func TestExtensionFilter_ShouldCompress(t *testing.T) {
	tests := []struct {
		name string
		req  *http.Request
		want bool
	}{
		{
			name: "no ext",
			req: &http.Request{
				URL:    mustParseURL("https://example.com/hello"),
				Method: http.MethodPost,
				Header: map[string][]string{"Accept-Encoding": {"gzip"}},
			},
			want: true,
		},
		{
			name: "txt",
			req: &http.Request{
				URL:    mustParseURL("https://example.com/a.txt"),
				Method: http.MethodPost,
				Header: map[string][]string{"Accept-Encoding": {"gzip"}},
			},
			want: true,
		},
		{
			name: "md",
			req: &http.Request{
				URL:    mustParseURL("https://example.com/a.txt.md"),
				Method: http.MethodPost,
				Header: map[string][]string{"Accept-Encoding": {"gzip"}},
			},
			want: true,
		},
		{
			name: "png",
			req: &http.Request{
				URL:    mustParseURL("https://example.com/a.exe.png"),
				Method: http.MethodPost,
				Header: map[string][]string{"Accept-Encoding": {"gzip"}},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := DefaultExtensionFilter()
			assert.Equal(t, tt.want, e.ShouldCompress(tt.req))
		})
	}
}

func mustParseURL(rawURL string) (URL *url.URL) {
	URL, err := url.Parse(rawURL)
	if err != nil {
		panic(err)
	}

	return
}
