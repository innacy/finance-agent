package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestBoxRendersContent(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)

	p.Box("Test Title", "Line 1\nLine 2")

	out := buf.String()
	if !strings.Contains(out, "Test Title") {
		t.Error("box should contain title")
	}
	if !strings.Contains(out, "Line 1") {
		t.Error("box should contain content line 1")
	}
	if !strings.Contains(out, "Line 2") {
		t.Error("box should contain content line 2")
	}
	if !strings.Contains(out, "╭") || !strings.Contains(out, "╰") {
		t.Error("box should have rounded corners")
	}
}

func TestTableRendersHeaderAndRows(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)

	headers := []string{"Name", "Amount", "Category"}
	rows := [][]string{
		{"Swiggy", "₹450", "Food"},
		{"Uber", "₹120", "Transport"},
	}

	p.Table(headers, rows)

	out := buf.String()
	if !strings.Contains(out, "Name") {
		t.Error("table should contain header 'Name'")
	}
	if !strings.Contains(out, "Swiggy") {
		t.Error("table should contain row data 'Swiggy'")
	}
	if !strings.Contains(out, "Uber") {
		t.Error("table should contain row data 'Uber'")
	}
}

func TestTableHandlesEmptyRows(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)

	headers := []string{"Name", "Amount"}
	p.Table(headers, [][]string{})

	out := buf.String()
	if !strings.Contains(out, "Name") {
		t.Error("empty table should still show headers")
	}
}

func TestSuccessPrintsCheckmark(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)

	p.Success("Connected")

	out := buf.String()
	if !strings.Contains(out, "✓") {
		t.Error("success should have checkmark")
	}
	if !strings.Contains(out, "Connected") {
		t.Error("success should contain message")
	}
}

func TestErrorPrintsCross(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)

	p.Error("Failed to connect")

	out := buf.String()
	if !strings.Contains(out, "✗") {
		t.Error("error should have cross mark")
	}
	if !strings.Contains(out, "Failed to connect") {
		t.Error("error should contain message")
	}
}

func TestWarnPrintsWarningSign(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)

	p.Warn("Low confidence")

	out := buf.String()
	if !strings.Contains(out, "⚠") {
		t.Error("warn should have warning sign")
	}
	if !strings.Contains(out, "Low confidence") {
		t.Error("warn should contain message")
	}
}

func TestInfoPrintsMessage(t *testing.T) {
	var buf bytes.Buffer
	p := NewPrinter(&buf, false)

	p.Info("Processing 12 transactions")

	out := buf.String()
	if !strings.Contains(out, "Processing 12 transactions") {
		t.Error("info should contain message")
	}
}
