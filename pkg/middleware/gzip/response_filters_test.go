package gzip

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSkipCompressedFilter_ShouldCompress(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
		want   bool
	}{
		{
			"should pass",
			make(http.Header),
			true,
		},
		{
			"gzip Content-Encoding",
			http.Header{"Content-Encoding": []string{"gzip"}},
			false,
		},
		{
			"br Content-Encoding",
			http.Header{"Content-Encoding": []string{"br"}},
			false,
		},
		{
			"complex Content-Encoding",
			http.Header{"Content-Encoding": []string{"deflate, gzip"}},
			false,
		},
		{
			"gzip Transfer-Encoding",
			http.Header{"Transfer-Encoding": []string{"gzip"}},
			false,
		},
		{
			"br Transfer-Encoding",
			http.Header{"Transfer-Encoding": []string{"br"}},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SkipCompressedFilter{}
			assert.Equal(t, tt.want, s.ShouldCompress(tt.header))
		})
	}
}

func TestContentTypeFilter_ShouldCompress(t *testing.T) {
	tests := []struct {
		header http.Header
		want   bool
	}{
		{
			contentTypeHeader(""),
			false,
		},
		{
			contentTypeHeader("application/json; charset=utf8"),
			true,
		},
		{
			contentTypeHeader("application/json"),
			true,
		},
		{
			contentTypeHeader("application/xml; charset=utf8"),
			true,
		},
		{
			contentTypeHeader("image/png"),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.header.Get("Content-Type"), func(t *testing.T) {
			d := DefaultContentTypeFilter()
			assert.Equal(t, tt.want, d.ShouldCompress(tt.header))
		})
	}
}

func TestContentTypeFilterEmpty_ShouldCompress(t *testing.T) {
	t.Run("empty content type is allowed", func(t *testing.T) {
		header := contentTypeHeader("")
		e := NewContentTypeFilter([]string{""})
		assert.Equal(t, true, e.ShouldCompress(header))
	})
}

func contentTypeHeader(contentType string) http.Header {
	return http.Header{"Content-Type": []string{contentType}}
}
