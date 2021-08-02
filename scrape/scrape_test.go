package scrape

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const exampleBody = `
<!DOCTYPE HTML>
<html>
<head>
	<title>Example Page</title>
</head>
<body>
	<h1>Hello <a href="https://google.com/">World!</a>
	<p>
		<a href="https://example.com/hello">Link 1</a>
	</p>
	<p>
		<a href="/blog">Blog</a>
	</p>
</body>
</html>
`

// Parses a url from a string or dies trying
func must(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}

	return u
}

func diffStrings(a, b []string) string {
	if len(a) != len(b) {
		return fmt.Sprintf("slices are not the same length, wanted: %#v, got: %#v", a, b)
	}

	errs := []string{}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			errs = append(errs, fmt.Sprintf("elements at index %d did not match, wanted: %s, got: %s", i, a[i], b[i]))
		}
	}

	if len(errs) > 1 {
		return strings.Join(errs, "\n")
	}

	return ""
}

func TestFetchLinksGetsAnchorTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte(exampleBody)); err != nil {
			t.Fatalf("error writing response: %s", err)
		}
	}))
	t.Cleanup(func() {
		server.Close()
	})
	u, _ := url.Parse(server.URL)
	client := &http.Client{Timeout: 5 * time.Second}
	scraper := NewScraper(client)
	expectedLinks := []string{
		"https://google.com",
		"https://example.com/hello",
		fmt.Sprintf("%s/blog", server.URL), // The host is local, so it'll get filled in as localhost
	}

	links, err := scraper.FetchLinks(u)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Create strings out of the urls to compare
	ss := []string{}
	for _, u := range links {
		ss = append(ss, u.String())
	}

	if diff := diffStrings(expectedLinks, ss); diff != "" {
		t.Fatalf("difference in expected links: %s", diff)
	}
}

func TestLinkToURL(t *testing.T) {
	table := []struct {
		actual   string
		expected *url.URL
	}{
		{"http://google.com/", must("http://google.com/")},
		{"/blog", must("https://example.com/blog")},
		{"https://www.example.com/news/", must("https://www.example.com/news/")},
		{"../", must("https://example.com/")},
		{"./products", must("https://example.com/news/products")},
	}

	base, _ := url.Parse("https://example.com/news/")

	for _, tt := range table {
		t.Run(
			fmt.Sprintf("actual: %s, expected: %s", tt.actual, tt.expected),
			func(t *testing.T) {
				u := linkToURL(base, tt.actual)
				if tt.expected == nil && u != nil {
					t.Errorf("expected nil, instead got: %s", u.String())
				}

				if u.String() != tt.expected.String() {
					t.Errorf("expected %s, instead got: %s", tt.expected.String(), u.String())
				}
			},
		)
	}
}
