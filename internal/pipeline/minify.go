package pipeline

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/html"
	"github.com/tdewolff/minify/v2/js"
)

var minifier = func() *minify.M {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("application/javascript", js.Minify)
	m.Add("text/html", &html.Minifier{KeepDocumentTags: true, KeepEndTags: true, KeepQuotes: true, KeepDefaultAttrVals: true})
	return m
}()

// minifyAsset minifies CSS or JS when enabled. Other files, and files that
// fail to parse, are returned unchanged.
func minifyAsset(name string, data []byte, cfg config.PipelineConfig) []byte {
	mediaType := ""
	switch strings.ToLower(filepath.Ext(name)) {
	case ".css":
		if cfg.MinifyCSS && !strings.HasSuffix(name, ".min.css") {
			mediaType = "text/css"
		}
	case ".js", ".mjs":
		if cfg.MinifyJS && !strings.HasSuffix(name, ".min.js") {
			mediaType = "application/javascript"
		}
	}
	if mediaType == "" {
		return data
	}
	var out bytes.Buffer
	if err := minifier.Minify(mediaType, &out, bytes.NewReader(data)); err != nil {
		return data
	}
	return out.Bytes()
}

// MinifyHTML minifies a rendered page, keeping whitespace that affects
// layout. On a parse error the page is returned unchanged.
func MinifyHTML(page string) string {
	var out bytes.Buffer
	if err := minifier.Minify("text/html", &out, strings.NewReader(page)); err != nil {
		return page
	}
	return strings.TrimSpace(out.String()) + "\n"
}
