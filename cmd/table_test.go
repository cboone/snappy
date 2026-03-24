package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestColumnTableLeftAlign(t *testing.T) {
	ct := &columnTable{
		cols: []columnDef{
			{title: "NAME", align: alignLeft},
			{title: "VALUE", align: alignLeft},
		},
	}
	ct.addRow("a", "short")
	ct.addRow("longer", "x")

	var buf bytes.Buffer
	if err := ct.render(&buf, true); err != nil {
		t.Fatalf("render() error = %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), buf.String())
	}
	// Header and data columns should align.
	if lines[0] != "  NAME    VALUE" {
		t.Errorf("header = %q, want %q", lines[0], "  NAME    VALUE")
	}
	if lines[1] != "  a       short" {
		t.Errorf("row 1 = %q, want %q", lines[1], "  a       short")
	}
	if lines[2] != "  longer  x" {
		t.Errorf("row 2 = %q, want %q", lines[2], "  longer  x")
	}
}

func TestColumnTableRightAlign(t *testing.T) {
	ct := &columnTable{
		cols: []columnDef{
			{title: "#", align: alignRight},
			{title: "COUNT", align: alignRight},
		},
	}
	ct.addRow("1", "42")
	ct.addRow("10", "7")

	var buf bytes.Buffer
	if err := ct.render(&buf, true); err != nil {
		t.Fatalf("render() error = %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), buf.String())
	}
	// Headers are always left-aligned, even for right-aligned columns.
	if lines[0] != "  #   COUNT" {
		t.Errorf("header = %q, want %q", lines[0], "  #   COUNT")
	}
	if lines[1] != "   1     42" {
		t.Errorf("row 1 = %q, want %q", lines[1], "   1     42")
	}
	if lines[2] != "  10      7" {
		t.Errorf("row 2 = %q, want %q", lines[2], "  10      7")
	}
}

func TestColumnTableMixedAlign(t *testing.T) {
	ct := &columnTable{
		cols: []columnDef{
			{title: "#", align: alignRight},
			{title: "NAME", align: alignLeft},
		},
	}
	ct.addRow("1", "alpha")
	ct.addRow("20", "b")

	var buf bytes.Buffer
	if err := ct.render(&buf, true); err != nil {
		t.Fatalf("render() error = %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), buf.String())
	}
	if lines[1] != "   1  alpha" {
		t.Errorf("row 1 = %q, want %q", lines[1], "   1  alpha")
	}
	if lines[2] != "  20  b" {
		t.Errorf("row 2 = %q, want %q", lines[2], "  20  b")
	}
}

func TestColumnTableNoHeader(t *testing.T) {
	ct := &columnTable{
		cols: []columnDef{
			{title: "NAME", align: alignLeft},
		},
	}
	ct.addRow("hello")

	var buf bytes.Buffer
	if err := ct.render(&buf, false); err != nil {
		t.Fatalf("render() error = %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "NAME") {
		t.Errorf("output should not contain header, got:\n%s", output)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("output should contain data, got:\n%s", output)
	}
}

func TestColumnTableHeaderOnly(t *testing.T) {
	ct := &columnTable{
		cols: []columnDef{
			{title: "COL1", align: alignLeft},
			{title: "COL2", align: alignLeft},
		},
	}

	var buf bytes.Buffer
	if err := ct.render(&buf, true); err != nil {
		t.Fatalf("render() error = %v", err)
	}

	output := strings.TrimRight(buf.String(), "\n")
	if output != "  COL1  COL2" {
		t.Errorf("output = %q, want %q", output, "  COL1  COL2")
	}
}

func TestColumnTableWidthFromHeader(t *testing.T) {
	ct := &columnTable{
		cols: []columnDef{
			{title: "LONGHEADER", align: alignLeft},
		},
	}
	ct.addRow("x")

	var buf bytes.Buffer
	if err := ct.render(&buf, true); err != nil {
		t.Fatalf("render() error = %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2:\n%s", len(lines), buf.String())
	}
	// Data cell should be padded to header width.
	if lines[1] != "  x" {
		t.Errorf("row = %q, want %q", lines[1], "  x")
	}
}
