package pipeline

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/MohamedElashri/nida/internal/route"
)

// pendingStylesheet is a stylesheet held back while fingerprinting until
// every other asset has its hashed name, so its references can point at
// those names and its own hash covers the bytes that are served.
type pendingStylesheet struct {
	rel  string
	data []byte
}

var (
	cssURLPattern    = regexp.MustCompile(`url\(\s*(?:"([^"]*)"|'([^']*)'|([^'")\s]+))\s*\)`)
	cssImportPattern = regexp.MustCompile(`@import\s+(?:"([^"]*)"|'([^']*)')`)
)

// finishStylesheets rewrites and fingerprints held-back stylesheets. A
// stylesheet that imports another waits for it, so the import can name the
// imported file's hash; an import cycle is written with what is known.
func finishStylesheets(outputRoot string, pending []pendingStylesheet, manifest Manifest, basePath string) error {
	waiting := make(map[string]bool, len(pending))
	for _, sheet := range pending {
		waiting[sheet.rel] = true
	}
	write := func(sheet pendingStylesheet) error {
		css := rewriteCSS(string(sheet.data), sheet.rel, manifest, basePath)
		fpPath, err := fingerprintData([]byte(css), sheet.rel, outputRoot)
		if err != nil {
			return fmt.Errorf("fingerprint %q: %w", sheet.rel, err)
		}
		manifest[sheet.rel] = fpPath
		delete(waiting, sheet.rel)
		return nil
	}
	for len(waiting) > 0 {
		progress := false
		for _, sheet := range pending {
			if !waiting[sheet.rel] || importsWaiting(sheet, waiting, basePath) {
				continue
			}
			if err := write(sheet); err != nil {
				return err
			}
			progress = true
		}
		if progress {
			continue
		}
		for _, sheet := range pending {
			if waiting[sheet.rel] {
				if err := write(sheet); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func importsWaiting(sheet pendingStylesheet, waiting map[string]bool, basePath string) bool {
	for _, ref := range cssRefs(string(sheet.data)) {
		if target, ok := cssTarget(ref, sheet.rel, basePath); ok && target != sheet.rel && waiting[target] {
			return true
		}
	}
	return false
}

// rewriteCSS points url() and @import references at fingerprinted names.
// cssRel is the stylesheet's path in the output, which relative references
// resolve against; site-relative references carry basePath.
func rewriteCSS(css, cssRel string, manifest Manifest, basePath string) string {
	replace := func(pattern *regexp.Regexp) {
		css = pattern.ReplaceAllStringFunc(css, func(match string) string {
			ref := firstGroup(pattern.FindStringSubmatch(match))
			if mapped, ok := mapCSSRef(ref, cssRel, manifest, basePath); ok {
				return strings.Replace(match, ref, mapped, 1)
			}
			return match
		})
	}
	replace(cssURLPattern)
	replace(cssImportPattern)
	return css
}

func cssRefs(css string) []string {
	var refs []string
	for _, pattern := range []*regexp.Regexp{cssURLPattern, cssImportPattern} {
		for _, m := range pattern.FindAllStringSubmatch(css, -1) {
			refs = append(refs, firstGroup(m))
		}
	}
	return refs
}

func firstGroup(submatches []string) string {
	for _, s := range submatches[1:] {
		if s != "" {
			return s
		}
	}
	return ""
}

// cssTarget resolves a stylesheet reference to an output-relative path.
func cssTarget(ref, cssRel, basePath string) (string, bool) {
	if ref == "" || strings.HasPrefix(ref, "#") || route.IsAbsoluteURL(ref) {
		return "", false
	}
	p, _ := route.SplitSuffix(ref)
	if strings.HasPrefix(p, "/") {
		return strings.CutPrefix(p, basePath+"/")
	}
	return path.Clean(path.Join(path.Dir(cssRel), p)), true
}

// mapCSSRef returns ref with its file name swapped for the fingerprinted
// one, keeping its directory part and any query or fragment. Fingerprinted
// files stay in the directory of their source, so only the name changes.
func mapCSSRef(ref, cssRel string, manifest Manifest, basePath string) (string, bool) {
	target, ok := cssTarget(ref, cssRel, basePath)
	if !ok {
		return "", false
	}
	mapped, ok := manifest[target]
	if !ok {
		return "", false
	}
	p, suffix := route.SplitSuffix(ref)
	return p[:strings.LastIndex(p, "/")+1] + path.Base(mapped) + suffix, true
}
