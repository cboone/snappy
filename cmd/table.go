package cmd

import (
	"fmt"
	"io"
	"strings"
)

type columnAlign int

const (
	alignLeft columnAlign = iota
	alignRight
)

type columnDef struct {
	title string
	align columnAlign
}

type columnTable struct {
	cols []columnDef
	rows [][]string
}

func (ct *columnTable) addRow(cells ...string) {
	if len(cells) != len(ct.cols) {
		panic(fmt.Sprintf("columnTable: got %d cells, want %d", len(cells), len(ct.cols)))
	}
	ct.rows = append(ct.rows, cells)
}

func (ct *columnTable) render(w io.Writer, showHeader bool) error {
	widths := make([]int, len(ct.cols))
	for i, col := range ct.cols {
		widths[i] = len(col.title)
	}
	for _, row := range ct.rows {
		for i, cell := range row {
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	if showHeader {
		if err := ct.writeHeader(w, widths); err != nil {
			return err
		}
	}
	for _, row := range ct.rows {
		if err := ct.writeRow(w, row, widths); err != nil {
			return err
		}
	}
	return nil
}

func (ct *columnTable) writeHeader(w io.Writer, widths []int) error {
	parts := make([]string, len(ct.cols))
	for i, col := range ct.cols {
		parts[i] = fmt.Sprintf("%-*s", widths[i], col.title)
	}
	line := strings.TrimRight("  "+strings.Join(parts, "  "), " ")
	_, err := fmt.Fprintln(w, line)
	return err
}

func (ct *columnTable) writeRow(w io.Writer, cells []string, widths []int) error {
	parts := make([]string, len(cells))
	for i, cell := range cells {
		if ct.cols[i].align == alignRight {
			parts[i] = fmt.Sprintf("%*s", widths[i], cell)
		} else {
			parts[i] = fmt.Sprintf("%-*s", widths[i], cell)
		}
	}
	// Trim trailing whitespace from the last column.
	line := strings.TrimRight("  "+strings.Join(parts, "  "), " ")
	_, err := fmt.Fprintln(w, line)
	return err
}
