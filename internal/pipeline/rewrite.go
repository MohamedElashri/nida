package pipeline

import (
	"regexp"
	"strings"
)

var rewritePatterns = []struct {
	pattern *regexp.Regexp
	attrs   []int
}{
	{regexp.MustCompile(`<link\b[^>]*\bhref="([^"]*\.css)"`), []int{1}},
	{regexp.MustCompile(`<script\b[^>]*\bsrc="([^"]*\.js)"`), []int{1}},
	{regexp.MustCompile(`<img\b[^>]*\bsrc="([^"]*(?:\.png|\.jpg|\.jpeg|\.gif|\.webp|\.svg))"`), []int{1}},
	{regexp.MustCompile(`<source\b[^>]*\bsrcset="([^"]*)"`), []int{1}},
	{regexp.MustCompile(`<link\b[^>]*\bhref="([^"]*\.svg)"`), []int{1}},
}

func RewriteHTML(html string, manifest Manifest) string {
	if len(manifest) == 0 {
		return html
	}

	for _, rp := range rewritePatterns {
		html = rp.pattern.ReplaceAllStringFunc(html, func(match string) string {
			submatches := rp.pattern.FindStringSubmatch(match)
			if len(submatches) < 2 {
				return match
			}
			for _, idx := range rp.attrs {
				if idx >= len(submatches) {
					continue
				}
				original := submatches[idx]
				mapped, ok := lookupManifest(manifest, original)
				if ok {
					return strings.Replace(match, original, mapped, 1)
				}
			}
			return match
		})
	}

	return html
}

func lookupManifest(manifest Manifest, path string) (string, bool) {
	path = strings.TrimPrefix(path, "/")

	if mapped, ok := manifest[path]; ok {
		return "/" + mapped, true
	}

	for original, mapped := range manifest {
		if strings.HasSuffix(path, original) {
			return "/" + mapped, true
		}
	}

	return "", false
}
