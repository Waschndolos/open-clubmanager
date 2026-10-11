package dsar

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestPDFIsWellFormed(t *testing.T) {
	long := strings.Repeat("Müller-Lüdenscheid (Straße) € ", 12)
	rep := &Report{GeneratedAt: time.Now(), GeneratedBy: "a@b.c", MemberID: "m1"}
	for i := 0; i < 8; i++ {
		rows := make([]Record, 20)
		for j := range rows {
			rows[j] = Record{{"k", "Name", long}, {"n", "Empty", nil}}
		}
		rep.Sections = append(rep.Sections, Section{Key: "s", Title: "Section", Records: rows})
	}
	out := rep.PDF()
	if !strings.HasPrefix(string(out), "%PDF-1.4") || !strings.HasSuffix(string(out), "%%EOF\n") {
		t.Fatal("not a PDF")
	}
	if strings.Count(string(out), "/Type /Page ") < 2 {
		t.Error("expected multiple pages")
	}
	if p := os.Getenv("DSAR_PDF_OUT"); p != "" {
		os.WriteFile(p, out, 0o600)
	}
}
