package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

type Printer struct {
	w     io.Writer
	color bool
}

func NewPrinter(w io.Writer, enableColor bool) *Printer {
	if !enableColor {
		color.NoColor = true
	}
	return &Printer{w: w, color: enableColor}
}

func (p *Printer) Box(title, content string) {
	lines := strings.Split(content, "\n")

	maxLen := len(title)
	for _, l := range lines {
		if len(l) > maxLen {
			maxLen = len(l)
		}
	}
	width := maxLen + 4

	fmt.Fprintf(p.w, "╭%s╮\n", strings.Repeat("─", width))
	fmt.Fprintf(p.w, "│  %-*s  │\n", maxLen, title)
	fmt.Fprintf(p.w, "├%s┤\n", strings.Repeat("─", width))
	for _, l := range lines {
		fmt.Fprintf(p.w, "│  %-*s  │\n", maxLen, l)
	}
	fmt.Fprintf(p.w, "╰%s╯\n", strings.Repeat("─", width))
}

func (p *Printer) Table(headers []string, rows [][]string) {
	colWidths := make([]int, len(headers))
	for i, h := range headers {
		colWidths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(colWidths) && len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	sep := "┼"
	headerLine := "│"
	divider := "├"
	topBorder := "┌"
	bottomBorder := "└"

	for i, w := range colWidths {
		topBorder += strings.Repeat("─", w+2)
		divider += strings.Repeat("─", w+2)
		bottomBorder += strings.Repeat("─", w+2)
		headerLine += fmt.Sprintf(" %-*s │", w, headers[i])
		if i < len(colWidths)-1 {
			topBorder += "┬"
			divider += sep
			bottomBorder += "┴"
		}
	}
	topBorder += "┐"
	divider += "┤"
	bottomBorder += "┘"

	fmt.Fprintln(p.w, topBorder)
	fmt.Fprintln(p.w, headerLine)
	fmt.Fprintln(p.w, divider)

	for _, row := range rows {
		line := "│"
		for i, w := range colWidths {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			line += fmt.Sprintf(" %-*s │", w, cell)
		}
		fmt.Fprintln(p.w, line)
	}

	fmt.Fprintln(p.w, bottomBorder)
}

func (p *Printer) Success(msg string) {
	fmt.Fprintf(p.w, "✓ %s\n", msg)
}

func (p *Printer) Error(msg string) {
	fmt.Fprintf(p.w, "✗ %s\n", msg)
}

func (p *Printer) Warn(msg string) {
	fmt.Fprintf(p.w, "⚠ %s\n", msg)
}

func (p *Printer) Info(msg string) {
	fmt.Fprintf(p.w, "  %s\n", msg)
}
