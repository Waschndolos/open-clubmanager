package dsar

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

const (
	pageW, pageH = 595, 842 // A4 in points
	margin       = 50
	wrapChars    = 92 // Helvetica 10pt is about 5.4pt per character on average
)

type pdfLine struct {
	text string
	size int
	bold bool
}

// winAnsi encodes s for a PDF literal string using WinAnsiEncoding.
func winAnsi(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r == '(' || r == ')' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r == '€':
			b.WriteByte(0x80)
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

func wrap(s string, n int) []string {
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

func display(v any) string {
	switch x := v.(type) {
	case nil:
		return "-"
	case *string:
		if x == nil || *x == "" {
			return "-"
		}
		return *x
	case string:
		if x == "" {
			return "-"
		}
		return x
	case float64:
		return fmt.Sprintf("%.2f", x)
	default:
		return fmt.Sprint(x)
	}
}

// PDF renders the report as a readable A4 document.
func (r *Report) PDF() []byte {
	var lines []pdfLine
	add := func(text string, size int, bold bool) {
		for _, l := range wrap(text, wrapChars*10/size) {
			lines = append(lines, pdfLine{l, size, bold})
		}
	}
	add("Data subject access report (GDPR Art. 15 / 20)", 16, true)
	add("Member ID: "+r.MemberID, 10, false)
	add("Created: "+r.GeneratedAt.Format(time.RFC3339)+" by "+r.GeneratedBy, 10, false)
	for _, s := range r.Sections {
		lines = append(lines, pdfLine{"", 10, false})
		add(s.Title, 13, true)
		if len(s.Records) == 0 {
			add("No data stored.", 10, false)
		}
		for i, rec := range s.Records {
			if i > 0 {
				lines = append(lines, pdfLine{"", 6, false})
			}
			for _, f := range rec {
				add(f.Label+": "+display(f.Value), 10, false)
			}
		}
	}

	// Paginate.
	var pages [][]string
	var cur strings.Builder
	y := pageH - margin
	started := false
	flush := func() {
		pages = append(pages, []string{cur.String()})
		cur.Reset()
		y = pageH - margin
		started = false
	}
	for _, l := range lines {
		step := l.size + 4
		if y-step < margin && started {
			flush()
		}
		y -= step
		started = true
		font := "F1"
		if l.bold {
			font = "F2"
		}
		fmt.Fprintf(&cur, "BT /%s %d Tf %d %d Td (%s) Tj ET\n", font, l.size, margin, y, winAnsi(l.text))
	}
	flush()

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
		objs = append(objs, fmt.Sprintf("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 %d %d] /Resources << /Font << /F1 3 0 R /F2 4 0 R >> >> /Contents %d 0 R >>", pageW, pageH, 6+2*i))
		objs = append(objs, fmt.Sprintf("<< /Length %d >>\nstream\n%sendstream", len(p[0]), p[0]))
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
