package markdown

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/MohamedElashri/nida/internal/config"
)

var (
	rawHTMLOpenRe  = regexp.MustCompile(`\{\{<\s*rawhtml\s*>\}\}`)
	rawHTMLCloseRe = regexp.MustCompile(`\{\{<\s*/rawhtml\s*>\}\}`)
	detailsStartRe = regexp.MustCompile(`\{%\s*details\s*\(([^}]*)\)\s*%\}`)
	summaryAttrRe  = regexp.MustCompile(`summary\s*=\s*"([^"]*)"`)
)

type shortcodeHandler func(args, body string, cfg config.SiteConfig) (string, error)

func blockShortcodeHandlers() map[string]shortcodeHandler {
	return map[string]shortcodeHandler{
		"details": renderDetailsShortcode,
	}
}

func processShortcodes(source string, cfg config.SiteConfig) (string, error) {
	source = rawHTMLOpenRe.ReplaceAllString(source, "")
	source = rawHTMLCloseRe.ReplaceAllString(source, "")
	return processDetailsShortcodes(source, cfg)
}

func processDetailsShortcodes(source string, cfg config.SiteConfig) (string, error) {
	var out strings.Builder
	remaining := source

	for {
		match := detailsStartRe.FindStringSubmatchIndex(remaining)
		if match == nil {
			out.WriteString(remaining)
			return out.String(), nil
		}

		out.WriteString(remaining[:match[0]])
		args := remaining[match[2]:match[3]]
		bodyStart := match[1]
		endStart, endEnd := findDetailsEnd(remaining[bodyStart:])
		if endStart < 0 {
			return "", fmt.Errorf("render markdown: unclosed details shortcode")
		}

		body := remaining[bodyStart : bodyStart+endStart]
		handler := blockShortcodeHandlers()["details"]
		rendered, err := handler(summaryValue(args), body, cfg)
		if err != nil {
			return "", err
		}

		out.WriteString(rendered)
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

func renderShortcodeBody(source string, cfg config.SiteConfig) (string, error) {
	processed, err := processShortcodes(source, cfg)
	if err != nil {
		return "", err
	}
	html, err := renderMarkdownCore(processed, cfg)
	if err != nil {
		return "", fmt.Errorf("render details shortcode body: %w", err)
	}
	return html, nil
}

func renderDetailsShortcode(summary, body string, cfg config.SiteConfig) (string, error) {
	renderedBody, err := renderShortcodeBody(body, cfg)
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
