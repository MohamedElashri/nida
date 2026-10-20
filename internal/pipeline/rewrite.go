package pipeline

import (
	"regexp"
	"strings"

	"github.com/MohamedElashri/nida/internal/route"
)

var rewritePatterns = []struct {
	pattern *regexp.Regexp
	attrs   []int
}{
	{regexp.MustCompile(`<link\b[^>]*\bhref="([^"]*\.css)"`), []int{1}},
	{regexp.MustCompile(`<script\b[^>]*\bsrc="([^"]*\.js)"`), []int{1}},
	{regexp.MustCompile(`<img\b[^>]*\bsrcset="([^"]*)"`), []int{1}},
	{regexp.MustCompile(`<img\b[^>]*\bsrc="([^"]*(?:\.png|\.jpg|\.jpeg|\.gif|\.webp|\.svg))"`), []int{1}},
	{regexp.MustCompile(`<source\b[^>]*\bsrcset="([^"]*)"`), []int{1}},
	{regexp.MustCompile(`<link\b[^>]*\bhref="([^"]*(?:\.svg|\.woff2?|\.ttf|\.eot))"`), []int{1}},
}

// rewriteHTML points asset references at their fingerprinted names.
// basePath is the base_url path, such as "/blog", that site-relative
// references start with.
func rewriteHTML(html string, manifest Manifest, basePath string) string {
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
				mapped, ok := lookupManifest(manifest, original, basePath)
				if ok {
					return strings.Replace(match, original, mapped, 1)
				}
				if rewritten, changed := rewriteSrcset(original, manifest, basePath); changed {
					return strings.Replace(match, original, rewritten, 1)
				}
			}
			return match
		})
	}

	return html
}

func lookupManifest(manifest Manifest, path, basePath string) (string, bool) {
	if route.IsAbsoluteURL(path) {
		return "", false
	}

	path, suffix := route.SplitSuffix(path)

	// Site-relative references carry the base path; others name the
	// asset directly.
	prefix := ""
	if rest, ok := strings.CutPrefix(path, basePath+"/"); ok && strings.HasPrefix(path, "/") {
		prefix, path = basePath+"/", rest
	}
	if mapped, ok := manifest[path]; ok {
		return prefix + mapped + suffix, true
	}
	return "", false
}

func rewriteSrcset(srcset string, manifest Manifest, basePath string) (string, bool) {
	parts := strings.Split(srcset, ",")
	changed := false
	for i, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		fields := strings.Fields(trimmed)
		if len(fields) == 0 {
			continue
		}
		if mapped, ok := lookupManifest(manifest, fields[0], basePath); ok {
			fields[0] = mapped
			parts[i] = strings.Join(fields, " ")
			changed = true
		}
	}
	if !changed {
		return srcset, false
	}
	return strings.Join(parts, ", "), true
}
