package clifmt

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"
)

const (
	defaultTableWidth     = 100
	defaultMinDetailWidth = 36
)

type NameDetailRow struct {
	Name   string
	Detail string
}

type NameDetailTableOptions struct {
	Title          string
	Rows           []NameDetailRow
	EmptyText      string
	IndexHeader    string
	NameHeader     string
	DetailHeader   string
	DefaultWidth   int
	MinDetailWidth int
	EmptyDetail    string
}

func PrintNameDetailTable(out io.Writer, opts NameDetailTableOptions) {
	if out == nil {
		out = os.Stdout
	}

	title := strings.TrimSpace(opts.Title)
	if title != "" {
		fmt.Fprintln(out, Headerf("%s (%d)", title, len(opts.Rows)))
	}

	if len(opts.Rows) == 0 {
		emptyText := strings.TrimSpace(opts.EmptyText)
		if emptyText == "" {
			emptyText = "No entries."
		}
		fmt.Fprintln(out, Warn(emptyText))
		return
	}

	indexHeader := strings.TrimSpace(opts.IndexHeader)
	if indexHeader == "" {
		indexHeader = "#"
	}
	nameHeader := strings.TrimSpace(opts.NameHeader)
	if nameHeader == "" {
		nameHeader = "NAME"
	}
	detailHeader := strings.TrimSpace(opts.DetailHeader)
	if detailHeader == "" {
		detailHeader = "DETAILS"
	}
	emptyDetail := strings.TrimSpace(opts.EmptyDetail)
	if emptyDetail == "" {
		emptyDetail = "No details provided."
	}

	indexWidth := utf8.RuneCountInString(indexHeader)
	maxIndexDigits := utf8.RuneCountInString(strconv.Itoa(len(opts.Rows) - 1))
	if maxIndexDigits > indexWidth {
		indexWidth = maxIndexDigits
	}

	nameWidth := utf8.RuneCountInString(nameHeader)
	for _, row := range opts.Rows {
		if width := utf8.RuneCountInString(row.Name); width > nameWidth {
			nameWidth = width
		}
	}

	detailWidth := tableDetailWidth(out, indexWidth, nameWidth, opts.DefaultWidth, opts.MinDetailWidth)

	fmt.Fprintf(out, "%s  %s  %s\n", Key(padRightRunes(indexHeader, indexWidth)), Key(padRightRunes(nameHeader, nameWidth)), Key(detailHeader))
	fmt.Fprintf(out, "%s  %s  %s\n", Dim(strings.Repeat("-", indexWidth)), Dim(strings.Repeat("-", nameWidth)), Dim(strings.Repeat("-", detailWidth)))

	for i, row := range opts.Rows {
		detail := strings.TrimSpace(row.Detail)
		if detail == "" {
			detail = emptyDetail
		}

		lines := wrapTextRunes(detail, detailWidth)
		fmt.Fprintf(out, "%s  %s  %s\n", Dim(padRightRunes(strconv.Itoa(i), indexWidth)), Success(padRightRunes(row.Name, nameWidth)), lines[0])
		for _, line := range lines[1:] {
			fmt.Fprintf(out, "%s  %s  %s\n", strings.Repeat(" ", indexWidth), strings.Repeat(" ", nameWidth), line)
		}
	}
}

func tableDetailWidth(out io.Writer, indexWidth, nameWidth, defaultWidth, minDetailWidth int) int {
	if defaultWidth <= 0 {
		defaultWidth = defaultTableWidth
	}
	if minDetailWidth <= 0 {
		minDetailWidth = defaultMinDetailWidth
	}

	width := defaultWidth
	if file, ok := out.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		if terminalWidth, _, err := term.GetSize(int(file.Fd())); err == nil && terminalWidth > 0 {
			width = terminalWidth
		}
	}

	detailWidth := width - indexWidth - nameWidth - 4
	if detailWidth < minDetailWidth {
		detailWidth = minDetailWidth
	}
	return detailWidth
}

func padRightRunes(s string, width int) string {
	missing := width - utf8.RuneCountInString(s)
	if missing <= 0 {
		return s
	}
	return s + strings.Repeat(" ", missing)
}

func wrapTextRunes(text string, width int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return []string{""}
	}
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	lines := make([]string, 0, len(words))
	current := ""

	flush := func() {
		if current == "" {
			return
		}
		lines = append(lines, current)
		current = ""
	}

	for _, word := range words {
		for utf8.RuneCountInString(word) > width {
			flush()
			runes := []rune(word)
			lines = append(lines, string(runes[:width]))
			word = string(runes[width:])
		}

		switch {
		case current == "":
			current = word
		case utf8.RuneCountInString(current)+1+utf8.RuneCountInString(word) <= width:
			current += " " + word
		default:
			flush()
			current = word
		}
	}
	flush()

	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}
