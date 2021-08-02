// Package filter provides the logic for looking at a url and matching
// it against the criteria for being "crawlable"
package filter

import (
	"net/url"
	"strings"
)

// Filter implements the logic for checking if a url is crawlable,
// maintaining a map to keep track of visited urls
type Filter struct {
	base       *url.URL
	baseDomain string
	visited    map[string]struct{}
}

// NewFilter creates a Filter with the visited map initialized.
// Also accepts a base url to compare all other urls against
func NewFilter(base *url.URL) *Filter {
	return &Filter{
		base:       base,
		baseDomain: baseDomain(*base),
		visited:    map[string]struct{}{},
	}
}

// Visit marks the url as visited, or if it doesn't pass criteria,
// returns false so that it can be skipped by the crawler.
// The criteria being:
// 1. The url hasn't been seen before
// 2. The url has the same domain
//
// It's debatable if queries or fragments should count as separate pages.
// Some pages use it as a ref tag, which means it's the same page,
// but analytics knows where you're coming from. On the other hand,
// some pages are rendered differently based on the query/fragment:
// https://google.com/blog?page=2. I'm making the deliberate choice that
// the are separate pages if they have different queries but not paths.
func (f *Filter) Visit(v *url.URL) bool {
	u := *v

	// Fix trailing slashes since they're the same page
	u.Path = strings.TrimSuffix(u.Path, "/")

	// Normalize the protocol, e.g. https:// should return the same as http://
	u.Scheme = "https"

	// Check that we haven't seen it before
	if _, ok := f.visited[u.String()]; ok {
		return false
	}

	// Check that it belongs to our base domain
	if f.baseDomain != baseDomain(u) {
		return false
	}

	// Mark is as seen now
	f.visited[u.String()] = struct{}{}

	return true
}

// Strips any subdomain and returns the main domain
func baseDomain(u url.URL) string {
	parts := strings.Split(u.Hostname(), ".")
	if len(parts) == 1 {
		return parts[0]
	}

	// Return the second to last piece
	return parts[len(parts)-2]
}
