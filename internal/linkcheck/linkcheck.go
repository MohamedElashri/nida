// Package linkcheck verifies external links found in rendered pages.
package linkcheck

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

var hrefPattern = regexp.MustCompile(`(?i)\bhref="(https?://[^"#]+)`)

// Source is one page's rendered HTML.
type Source struct {
	Path string
	HTML string
}

// Problem is a link that could not be fetched or returned an error status.
type Problem struct {
	URL     string
	Pages   []string
	Message string
}

// Collect returns external http(s) links mapped to the pages that use them.
func Collect(sources []Source) map[string][]string {
	links := map[string][]string{}
	for _, source := range sources {
		for _, match := range hrefPattern.FindAllStringSubmatch(source.HTML, -1) {
			url := strings.ReplaceAll(match[1], "&amp;", "&")
			if !slices.Contains(links[url], source.Path) {
				links[url] = append(links[url], source.Path)
			}
		}
	}
	return links
}

// Check requests every link with a few concurrent workers. It tries HEAD
// first and falls back to GET, since some servers reject HEAD.
func Check(ctx context.Context, links map[string][]string, client *http.Client) []Problem {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	urls := make([]string, 0, len(links))
	for url := range links {
		urls = append(urls, url)
	}
	slices.Sort(urls)

	var (
		mu       sync.Mutex
		problems []Problem
		wg       sync.WaitGroup
	)
	queue := make(chan string)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range queue {
				if msg := probe(ctx, client, url); msg != "" {
					mu.Lock()
					problems = append(problems, Problem{URL: url, Pages: links[url], Message: msg})
					mu.Unlock()
				}
			}
		}()
	}
	for _, url := range urls {
		queue <- url
	}
	close(queue)
	wg.Wait()

	slices.SortFunc(problems, func(a, b Problem) int { return strings.Compare(a.URL, b.URL) })
	return problems
}

func probe(ctx context.Context, client *http.Client, url string) string {
	status, err := request(ctx, client, http.MethodHead, url)
	if err != nil || status == http.StatusMethodNotAllowed || status == http.StatusForbidden || status == http.StatusNotImplemented {
		status, err = request(ctx, client, http.MethodGet, url)
	}
	if err != nil {
		return err.Error()
	}
	if status >= 400 {
		return fmt.Sprintf("HTTP %d", status)
	}
	return ""
}

func request(ctx context.Context, client *http.Client, method, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "nida-linkcheck (+https://github.com/MohamedElashri/nida)")
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	_ = resp.Body.Close()
	return resp.StatusCode, nil
}
