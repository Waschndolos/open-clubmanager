package dsar

import (
	"fmt"
	"time"

	"github.com/Waschndolos/open-clubmanager/server/internal/pdfdoc"
)

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
	d := pdfdoc.New(false)
	d.Line(16, true, "Data subject access report (GDPR Art. 15 / 20)")
	d.Line(10, false, "Member ID: "+r.MemberID)
	d.Line(10, false, "Created: "+r.GeneratedAt.Format(time.RFC3339)+" by "+r.GeneratedBy)
	for _, s := range r.Sections {
		d.Row(10, false)
		d.Line(13, true, s.Title)
		if len(s.Records) == 0 {
			d.Line(10, false, "No data stored.")
		}
		for i, rec := range s.Records {
			if i > 0 {
				d.Row(6, false)
			}
			for _, f := range rec {
				d.Line(10, false, f.Label+": "+display(f.Value))
			}
		}
	}
	return d.Bytes()
}
