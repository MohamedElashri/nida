package markdown

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
)

var (
	rawHTMLOpenRe  = regexp.MustCompile(`\{\{<\s*rawhtml\s*>\}\}`)
	rawHTMLCloseRe = regexp.MustCompile(`\{\{<\s*/rawhtml\s*>\}\}`)
	detailsStartRe = regexp.MustCompile(`\{%\s*details\s*\(([^}]*)\)\s*%\}`)
	summaryAttrRe  = regexp.MustCompile(`summary\s*=\s*"([^"]*)"`)
)

type shortcodeHandler func(args, body string, cfg config.SiteConfig, pathLookup PathLookup) (string, error)

type shortcodeResult struct {
	Source       string
	Replacements map[string]string
}

func blockShortcodeHandlers() map[string]shortcodeHandler {
	return map[string]shortcodeHandler{
		"details": renderDetailsShortcode,
	}
}

func processShortcodes(source string, cfg config.SiteConfig, pathLookup PathLookup) (shortcodeResult, error) {
	source = rawHTMLOpenRe.ReplaceAllString(source, "")
	source = rawHTMLCloseRe.ReplaceAllString(source, "")
	return processDetailsShortcodes(source, cfg, pathLookup)
}

func processDetailsShortcodes(source string, cfg config.SiteConfig, pathLookup PathLookup) (shortcodeResult, error) {
	var out strings.Builder
	remaining := source
	replacements := map[string]string{}

	for {
		match := detailsStartRe.FindStringSubmatchIndex(remaining)
		if match == nil {
			out.WriteString(remaining)
			return shortcodeResult{Source: out.String(), Replacements: replacements}, nil
		}

		out.WriteString(remaining[:match[0]])
		args := remaining[match[2]:match[3]]
		bodyStart := match[1]
		endStart, endEnd := findDetailsEnd(remaining[bodyStart:])
		if endStart < 0 {
			return shortcodeResult{}, fmt.Errorf("render markdown: unclosed details shortcode")
		}

		body := remaining[bodyStart : bodyStart+endStart]
		handler := blockShortcodeHandlers()["details"]
		rendered, err := handler(summaryValue(args), body, cfg, pathLookup)
		if err != nil {
			return shortcodeResult{}, err
		}

		placeholder := "@@NIDA_SHORTCODE_HTML_" + strconv.Itoa(len(replacements)) + "@@"
		replacements[placeholder] = rendered
		out.WriteString("\n\n")
		out.WriteString(placeholder)
		out.WriteString("\n\n")
		remaining = remaining[bodyStart+endEnd:]
	}
}

func findDetailsEnd(source string) (int, int) {
	re := regexp.MustCompile(`\{%\s*end\s*%\}`)
	match := re.FindStringIndex(source)
	if match == nil {
		return -1, -1
	}
	return match[0], match[1]
}

func renderShortcodeBody(source string, cfg config.SiteConfig, pathLookup PathLookup) (string, error) {
	processed, err := processShortcodes(source, cfg, pathLookup)
	if err != nil {
		return "", err
	}
	html, err := renderMarkdownCore(processed.Source, cfg, pathLookup)
	if err != nil {
		return "", fmt.Errorf("render details shortcode body: %w", err)
	}
	return restoreShortcodeHTML(html, processed.Replacements), nil
}

func renderDetailsShortcode(summary, body string, cfg config.SiteConfig, pathLookup PathLookup) (string, error) {
	renderedBody, err := renderShortcodeBody(body, cfg, pathLookup)
	if err != nil {
		return "", err
	}
	return renderDetails(summary, renderedBody), nil
}

func summaryValue(args string) string {
	match := summaryAttrRe.FindStringSubmatch(args)
	if match == nil {
		return "Show details"
	}
	return strings.TrimSpace(match[1])
}

func renderDetails(summary, body string) string {
	var b strings.Builder
	b.WriteString("\n<details class=\"collapsible-details\">\n")
	b.WriteString("  <summary class=\"collapsible-details-summary\">\n")
	b.WriteString("    <span class=\"collapsible-details-icon\"></span>\n")
	b.WriteString("    <span class=\"collapsible-details-label\">")
	b.WriteString(html.EscapeString(summary))
	b.WriteString("</span>\n")
	b.WriteString("  </summary>\n")
	b.WriteString("  <div class=\"collapsible-details-body\">\n")
	b.WriteString(body)
	b.WriteString("  </div>\n")
	b.WriteString("</details>\n")
	return b.String()
}

func restoreShortcodeHTML(rendered string, replacements map[string]string) string {
	for placeholder, replacement := range replacements {
		rendered = strings.ReplaceAll(rendered, "<p>"+placeholder+"</p>\n", replacement)
		rendered = strings.ReplaceAll(rendered, "<p>"+placeholder+"</p>", replacement)
		rendered = strings.ReplaceAll(rendered, placeholder, replacement)
	}
	return rendered
}
