// Package scrape provides functionality for getting link information
// from a web page via a url
package scrape

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/net/html"
)

// ErrorBadResponse occurs when fetching a page returns a non-2XX code
type ErrorBadResponse struct {
	StatusCode int
	Msg        string
}

// Error implements the error interface, formatting a message about the
// error code and error body
func (e ErrorBadResponse) Error() string {
	return fmt.Sprintf("received status %d with message %s", e.StatusCode, e.Msg)
}

// Scraper holds the dependencies for fetching webpages
type Scraper struct {
	client *http.Client
}

// NewScraper creates a new scraper struct with the provided
// *http.Client, which should be configured with a proper timeout
// by the provider.
func NewScraper(client *http.Client) Scraper {
	return Scraper{
		client: client,
	}
}

// FetchLinks goes to a url and parses it to return all links that
// are from the same domain.
// If a non-200 is returned, it returns an error, or if another
// error occurs.
func (s Scraper) FetchLinks(u *url.URL) ([]*url.URL, error) {
	// Perform the request to the url, and get the body
	resp, err := s.client.Get(u.String())
	if err != nil {
		return nil, fmt.Errorf("error getting %s: %s", u, err)
	}
	defer resp.Body.Close()

	// Check for a 2xx response code
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("non 2xx code returned fetching %s: %d", u, err)
	}

	// Parse the html
	links, err := parseAnchorLinks(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error parsing content of %s: %s", u, err)
	}

	// Fill in the domain for any path links and convert them all to urls
	filled := []*url.URL{}
	for _, l := range links {
		s := linkToURL(u, l)
		if s == nil {
			continue
		}

		filled = append(filled, s)
	}

	return filled, nil
}

// Parses an html body and returns all anchor tag href's
func parseAnchorLinks(r io.Reader) ([]string, error) {
	root, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("error parsing html: %s", err)
	}

	// Defining an inline recursive function for breadth-first traversal
	var f func(*html.Node) []string
	f = func(n *html.Node) []string {
		as := []string{}

		// If the node is an anchor tag, pull its href attribute value and add it
		// to the strings to be returned
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					as = append(as, a.Val)
					break
				}
			}
		}

		// Loop through the next layer of children and this node's siblings
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			as = append(as, f(c)...)
		}

		return as
	}

	return f(root), nil
}

// Takes a host and fills it in for any paths missing it.
// For example, a valid href could be `/blog`, but in reality refers to
// https://{domain}/blog. Also deals with relative urls, so things like ../
// at https://google.com/blog is referring to -> https://google.com
func linkToURL(reqURL *url.URL, link string) *url.URL {
	// For each path, parse it into a url to check if it needs a hostname
	// filled in. For paths that cannot be parsed, they're a misformmated
	// link and can be skipped.
	p, err := url.Parse(link)
	if err != nil {
		return nil
	}

	// Handles relative links like "../"
	resolved := reqURL.ResolveReference(p)

	return resolved
}
