// Package pdfdoc writes simple text documents as PDF: lines of text, optionally
// in columns, paginated on A4 with the standard Helvetica fonts. It needs no
// external dependency and no embedded font; text is encoded as WinAnsi, which
// covers German and French.
package pdfdoc

import (
	"bytes"
	"fmt"
	"strings"
)

const (
	a4W, a4H = 595, 842 // A4 in points
	// Margin is the page margin in points.
	Margin = 50
)

// Cell is a piece of text at horizontal offset X (points from the left margin).
type Cell struct {
	X    int
	Text string
}

// Doc collects rows and renders them into pages.
type Doc struct {
	w, h    int
	pages   []string
	cur     strings.Builder
	y       int
	started bool
}

// New returns an empty A4 document, in landscape orientation if requested.
func New(landscape bool) *Doc {
	d := &Doc{w: a4W, h: a4H}
	if landscape {
		d.w, d.h = a4H, a4W
	}
	d.y = d.h - Margin
	return d
}

// Width is the usable width between the margins.
func (d *Doc) Width() int { return d.w - 2*Margin }

// Row writes one line in the given font size; empty cells only take space.
func (d *Doc) Row(size int, bold bool, cells ...Cell) {
	step := size + 4
	if d.y-step < Margin && d.started {
		d.NewPage()
	}
	d.y -= step
	d.started = true
	font := "F1"
	if bold {
		font = "F2"
	}
	for _, c := range cells {
		if c.Text == "" {
			continue
		}
		fmt.Fprintf(&d.cur, "BT /%s %d Tf %d %d Td (%s) Tj ET\n", font, size, Margin+c.X, d.y, WinAnsi(c.Text))
	}
}

// Line writes text at the left margin, wrapped to the page width.
func (d *Doc) Line(size int, bold bool, text string) {
	// Helvetica averages about 0.54 em per character.
	chars := d.Width() * 100 / (size * 54)
	for _, l := range Wrap(text, chars) {
		d.Row(size, bold, Cell{Text: l})
	}
}

// NeedSpace starts a new page unless the given number of rows of the given
// size still fit, so that e.g. a heading is not left alone at a page end.
func (d *Doc) NeedSpace(rows, size int) {
	if d.started && d.y-rows*(size+4) < Margin {
		d.NewPage()
	}
}

// NewPage ends the current page.
func (d *Doc) NewPage() {
	d.pages = append(d.pages, d.cur.String())
	d.cur.Reset()
	d.y = d.h - Margin
	d.started = false
}

// Bytes renders the document.
func (d *Doc) Bytes() []byte {
	pages := append(append([]string(nil), d.pages...), d.cur.String())

	// Objects: 1 catalog, 2 pages, 3 F1, 4 F2, then page/content pairs.
	var objs []string
	objs = append(objs, "<< /Type /Catalog /Pages 2 0 R >>")
	kids := make([]string, len(pages))
	for i := range pages {
		kids[i] = fmt.Sprintf("%d 0 R", 5+2*i)
	}
	objs = append(objs, fmt.Sprintf("<< /Type /Pages /Kids [%s] /Count %d >>", strings.Join(kids, " "), len(pages)))
	objs = append(objs, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>")
	objs = append(objs, "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica-Bold /Encoding /WinAnsiEncoding >>")
	for i, p := range pages {
		objs = append(objs, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>", d.w, d.h, 6+2*i))
		objs = append(objs, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(p), p))
	}

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs))
	for i, o := range objs {
		offsets[i] = buf.Len()
		fmt.Fprintf(&buf, "%d 0 obj\n%s\nendobj\n", i+1, o)
	}
	xref := buf.Len()
	fmt.Fprintf(&buf, "xref\n0 %d\n0000000000 65535 f \n", len(objs)+1)
	for _, off := range offsets {
		fmt.Fprintf(&buf, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&buf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return buf.Bytes()
}

// WinAnsi encodes s for a PDF literal string using WinAnsiEncoding.
func WinAnsi(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '(' || r == ')' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '€':
			b.WriteByte(0x80)
		case r == '–':
			b.WriteByte(0x96)
		case r == '\n' || r == '\r' || r == '\t':
			b.WriteByte(' ')
		case r >= 0x20 && r < 0x7f, r >= 0xa0 && r <= 0xff:
			if r < 0x80 {
				b.WriteRune(r)
			} else {
				b.WriteByte(byte(r))
			}
		default:
			b.WriteByte('?')
		}
	}
	return b.String()
}

// Wrap breaks s into lines of at most n characters, at spaces where possible.
func Wrap(s string, n int) []string {
	r := []rune(s)
	var out []string
	for len(r) > n {
		cut := n
		for i := n; i > 0; i-- {
			if r[i] == ' ' {
				cut = i
				break
			}
		}
		out = append(out, strings.TrimRight(string(r[:cut]), " "))
		r = []rune(strings.TrimLeft(string(r[cut:]), " "))
	}
	return append(out, string(r))
}

// Fit shortens s to at most n characters for a table cell, ending with "...".
func Fit(s string, n int) string {
	r := []rune(s)
	if len(r) <= n || n < 4 {
		return s
	}
	return string(r[:n-3]) + "..."
}
