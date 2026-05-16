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
	{regexp.MustCompile(`<img\b[^>]*\bsrcset="([^"]*)"`), []int{1}},
	{regexp.MustCompile(`<img\b[^>]*\bsrc="([^"]*(?:\.png|\.jpg|\.jpeg|\.gif|\.webp|\.svg))"`), []int{1}},
	{regexp.MustCompile(`<source\b[^>]*\bsrcset="([^"]*)"`), []int{1}},
	{regexp.MustCompile(`<link\b[^>]*\bhref="([^"]*(?:\.svg|\.woff2?|\.ttf|\.eot))"`), []int{1}},
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
				if rewritten, changed := rewriteSrcset(original, manifest); changed {
					return strings.Replace(match, original, rewritten, 1)
				}
			}
			return match
		})
	}

	return html
}

func lookupManifest(manifest Manifest, path string) (string, bool) {
	if isExternalAssetPath(path) {
		return "", false
	}

	suffixIndex := len(path)
	for _, marker := range []string{"?", "#"} {
		if index := strings.Index(path, marker); index >= 0 && index < suffixIndex {
			suffixIndex = index
		}
	}
	suffix := path[suffixIndex:]
	path = path[:suffixIndex]
	leadingSlash := strings.HasPrefix(path, "/")
	path = strings.TrimPrefix(path, "/")

	if mapped, ok := manifest[path]; ok {
		return leadingAssetSlash(leadingSlash) + mapped + suffix, true
	}

	for original, mapped := range manifest {
		if strings.HasSuffix(path, original) {
			prefix := strings.TrimSuffix(path, original)
			return leadingAssetSlash(leadingSlash) + prefix + mapped + suffix, true
		}
	}

	return "", false
}

func rewriteSrcset(srcset string, manifest Manifest) (string, bool) {
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
		if mapped, ok := lookupManifest(manifest, fields[0]); ok {
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

func isExternalAssetPath(path string) bool {
	return strings.Contains(path, "://") || strings.HasPrefix(path, "//")
}

func leadingAssetSlash(leading bool) string {
	if leading {
		return "/"
	}
	return ""
}
