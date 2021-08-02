package filter_test

import (
	"net/url"
	"testing"

	"github.com/ankored/crawl/filter"
)

func TestVisit(t *testing.T) {
	table := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			"trailing slashes",
			"https://example.com/",
			false,
		},
		{
			"same path, different subdomain",
			"https://sub.example.com/news",
			true,
		},
		{
			"already visited url",
			"https://sub.example.com",
			false,
		},
		{
			"base domain is different",
			"https://google.com",
			false,
		},
		{
			"protocol is different",
			"http://example.com/news",
			false,
		},
		{
			"has not be visited",
			"https://example.com/products",
			true,
		},
	}

	// Sets up a filter with already visited urls
	setupFilter := func() *filter.Filter {
		base, _ := url.Parse("https://example.com")
		f := filter.NewFilter(base)
		f.Visit(base)

		u, _ := url.Parse("https://sub.example.com/")
		f.Visit(u)

		u, _ = url.Parse("https://example.com/news")
		f.Visit(u)

		return f
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {
			f := setupFilter()
			u, _ := url.Parse(tt.input)
			if got := f.Visit(u); got != tt.expected {
				t.Errorf("expected %v, instead got: %v", tt.expected, got)
			}
		})
	}
}
