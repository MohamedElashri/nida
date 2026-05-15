package cli

import (
	"io"
	"os"
	"strings"
)

type terminalColors struct {
	enabled bool
}

const (
	colorReset       = "\x1b[0m"
	colorBoldCyan    = "\x1b[1;36m"
	colorBoldGreen   = "\x1b[1;32m"
	colorBoldRed     = "\x1b[1;31m"
	colorBoldYellow  = "\x1b[1;33m"
	colorBoldMagenta = "\x1b[1;35m"
)

func colorsFor(w io.Writer) terminalColors {
	return terminalColors{enabled: writerSupportsColor(w)}
}

func writerSupportsColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}

	file, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func (c terminalColors) command(value string) string {
	return c.paint(colorBoldCyan, value)
}

func (c terminalColors) success(value string) string {
	return c.paint(colorBoldGreen, value)
}

func (c terminalColors) error(value string) string {
	return c.paint(colorBoldRed, value)
}

func (c terminalColors) warning(value string) string {
	return c.paint(colorBoldYellow, value)
}

func (c terminalColors) highlight(value string) string {
	return c.paint(colorBoldMagenta, value)
}

func (c terminalColors) paint(style, value string) string {
	if !c.enabled {
		return value
	}
	return style + value + colorReset
}
