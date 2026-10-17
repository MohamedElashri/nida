package pipeline

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

var minifier = func() *minify.M {
	m := minify.New()
	m.AddFunc("text/css", css.Minify)
	m.AddFunc("application/javascript", js.Minify)
	return m
}()

// minifyAsset returns minified bytes for CSS and JS files when the matching
// pipeline option is on. Other files, and files the minifier cannot parse,
// are returned unchanged so a minifier problem never breaks a build.
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
