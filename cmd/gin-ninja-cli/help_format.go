package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiCyan   = "\x1b[36m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
)

var helpColorEnabled = shouldUseHelpColor

type helpPrinter struct {
	w     io.Writer
	color bool
}

type helpItem struct {
	name  string
	usage string
}

func newHelpPrinter(w io.Writer) helpPrinter {
	return helpPrinter{
		w:     w,
		color: helpColorEnabled(w),
	}
}

func shouldUseHelpColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("CI") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
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

func (p helpPrinter) title(text string) string {
	return p.style(text, ansiBold, ansiCyan)
}

func (p helpPrinter) section(text string) string {
	return p.style(strings.ToUpper(text), ansiBold)
}

func (p helpPrinter) command(text string) string {
	return p.style(text, ansiGreen)
}

func (p helpPrinter) note(text string) string {
	return p.style(text, ansiYellow)
}

func (p helpPrinter) style(text string, codes ...string) string {
	if !p.color {
		return text
	}
	return strings.Join(codes, "") + text + ansiReset
}

func printHelpItems(w io.Writer, style func(string) string, items []helpItem) {
	width := 0
	for _, item := range items {
		if len(item.name) > width {
			width = len(item.name)
		}
	}
	for _, item := range items {
		fmt.Fprintf(w, "  %s  %s\n", style(fmt.Sprintf("%-*s", width, item.name)), item.usage)
	}
}
