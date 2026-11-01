package donation

import (
	"strings"

	"github.com/Waschndolos/open-clubmanager/server/internal/pdfdoc"
)

const (
	body  = 10
	small = 8
)

// The legal texts follow the official templates (BMF, Muster für
// Zuwendungsbestätigungen); only the applicable exemption is printed.
const (
	titleSingle     = "Bestätigung über Geldzuwendungen/Mitgliedsbeitrag"
	titleCollective = "Sammelbestätigung über Geldzuwendungen/Mitgliedsbeiträge"
	legalBasis      = "im Sinne des § 10b des Einkommensteuergesetzes an eine der in § 5 Abs. 1 Nr. 9 des Körperschaftsteuergesetzes " +
		"bezeichneten Körperschaften, Personenvereinigungen oder Vermögensmassen"
	notMembershipFee = "Es wird bestätigt, dass es sich nicht um einen Mitgliedsbeitrag handelt, dessen Abzug nach § 10b Abs. 1 " +
		"des Einkommensteuergesetzes ausgeschlossen ist."
	noFurtherReceipts = "Es wird bestätigt, dass über die in der Gesamtsumme enthaltenen Zuwendungen keine weiteren Bestätigungen, " +
		"weder formelle Zuwendungsbestätigungen noch Beitragsquittungen oder Ähnliches ausgestellt wurden und werden."
	waiverInAnnex = "Ob es sich um den Verzicht auf Erstattung von Aufwendungen handelt, ist der Anlage zur Sammelbestätigung zu entnehmen."
	liability     = "Hinweis: Wer vorsätzlich oder grob fahrlässig eine unrichtige Zuwendungsbestätigung erstellt oder veranlasst, " +
		"dass Zuwendungen nicht zu den in der Zuwendungsbestätigung angegebenen steuerbegünstigten Zwecken verwendet werden, " +
		"haftet für die entgangene Steuer (§ 10b Abs. 4 EStG, § 9 Abs. 3 KStG, § 9 Nr. 5 GewStG)."
	recognition = "Diese Bestätigung wird nicht als Nachweis für die steuerliche Berücksichtigung der Zuwendung anerkannt, " +
		"wenn das Datum des Freistellungsbescheides länger als 5 Jahre bzw. das Datum der Feststellung der Einhaltung der " +
		"satzungsmäßigen Voraussetzungen nach § 60a Abs. 1 AO länger als 3 Jahre seit Ausstellung der Bestätigung " +
		"zurückliegt (§ 63 Abs. 5 AO)."
)

const dateFormat = "02.01.2006"

func check(on bool) string {
	if on {
		return "[X]"
	}
	return "[  ]"
}

// PDF renders receipts into one document, each starting on a new page.
func PDF(receipts []Receipt) []byte {
	d := pdfdoc.New(false)
	for i, r := range receipts {
		if i > 0 {
			d.NewPage()
		}
		r.write(d)
	}
	return d.Bytes()
}

// exemptionText is the paragraph naming the tax exemption.
func (i Issuer) exemptionText() string {
	date := i.ExemptionDate.Format(dateFormat)
	if i.ExemptionKind == ExemptionAssessment {
		return "Die Einhaltung der satzungsmäßigen Voraussetzungen nach den §§ 51, 59, 60 und 61 AO wurde vom Finanzamt " +
			i.TaxOffice + ", StNr. " + i.TaxNumber + ", mit Bescheid vom " + date +
			" nach § 60a AO gesondert festgestellt. Wir fördern nach unserer Satzung " + i.Purposes + "."
	}
	period := i.AssessmentPeriod
	if period == "" {
		period = "-"
	}
	return "Wir sind wegen Förderung " + i.Purposes + " nach dem Freistellungsbescheid bzw. nach der Anlage zum " +
		"Körperschaftsteuerbescheid des Finanzamtes " + i.TaxOffice + ", StNr. " + i.TaxNumber + ", vom " + date +
		" für den letzten Veranlagungszeitraum " + period + " nach § 5 Abs. 1 Nr. 9 des Körperschaftsteuergesetzes von der " +
		"Körperschaftsteuer und nach § 3 Nr. 6 des Gewerbesteuergesetzes von der Gewerbesteuer befreit."
}

func (r Receipt) write(d *pdfdoc.Doc) {
	collective := r.Kind == KindCollective
	d.Row(small, false, pdfdoc.Cell{Text: "Aussteller (Bezeichnung und Anschrift der steuerbegünstigten Einrichtung)"})
	d.Row(body, true, pdfdoc.Cell{Text: r.Issuer.Name})
	for _, l := range lines(r.Issuer.Address) {
		d.Row(body, false, pdfdoc.Cell{Text: l})
	}
	d.Row(body, false)

	r.writeMarks(d)
	title := titleSingle
	if collective {
		title = titleCollective
	}
	d.Row(13, true, pdfdoc.Cell{Text: title})
	d.Line(small, false, legalBasis)
	d.Row(body, false, pdfdoc.Cell{Text: "Lfd. Nr. " + r.Number})
	d.Row(body, false)

	d.Row(small, false, pdfdoc.Cell{Text: "Name und Anschrift des Zuwendenden:"})
	d.Row(body, true, pdfdoc.Cell{Text: r.RecipientName})
	for _, l := range lines(r.RecipientAddress) {
		d.Row(body, false, pdfdoc.Cell{Text: l})
	}
	d.Row(body, false)

	amountLabel, dateLabel := "Betrag der Zuwendung - in Ziffern -", "Tag der Zuwendung:"
	dateValue := r.PeriodStart.Format(dateFormat)
	if collective {
		amountLabel, dateLabel = "Gesamtbetrag der Zuwendung - in Ziffern -", "Zeitraum der Sammelbestätigung:"
		dateValue = r.PeriodStart.Format(dateFormat) + " - " + r.PeriodEnd.Format(dateFormat)
	}
	d.Row(small, false, pdfdoc.Cell{Text: amountLabel}, pdfdoc.Cell{X: 170, Text: "- in Buchstaben -"},
		pdfdoc.Cell{X: 370, Text: dateLabel})
	words := pdfdoc.Wrap("- "+AmountWords(r.TotalCents())+" -", 36)
	for i, w := range words {
		cells := []pdfdoc.Cell{{X: 170, Text: w}}
		if i == 0 {
			cells = append(cells, pdfdoc.Cell{Text: Amount(r.TotalCents())}, pdfdoc.Cell{X: 370, Text: dateValue})
		}
		d.Row(body, i == 0, cells...)
	}
	d.Row(body, false)

	if collective {
		d.Line(body, false, waiverInAnnex)
	} else {
		waiver := len(r.Items) > 0 && r.Items[0].ExpenseWaiver
		d.Row(body, false, pdfdoc.Cell{Text: "Es handelt sich um den Verzicht auf Erstattung von Aufwendungen:"},
			pdfdoc.Cell{X: 330, Text: "Ja " + check(waiver)}, pdfdoc.Cell{X: 380, Text: "Nein " + check(!waiver)})
	}
	d.Row(body, false)

	d.Line(body, false, check(true)+" "+r.Issuer.exemptionText())
	d.Row(body, false)
	use := "Es wird bestätigt, dass die Zuwendung nur zur Förderung " + r.Issuer.Purposes + " verwendet wird."
	if collective {
		use = "Es wird bestätigt, dass die Zuwendungen nur zur Förderung " + r.Issuer.Purposes + " verwendet werden."
	}
	d.Line(body, false, use)
	if !r.Issuer.FeesDeductible {
		d.Row(body, false)
		d.Line(small, false, "Nur für steuerbegünstigte Einrichtungen, bei denen die Mitgliedsbeiträge steuerlich nicht abziehbar sind:")
		d.Line(body, false, check(true)+" "+notMembershipFee)
	}
	if collective {
		d.Row(body, false)
		d.Line(body, false, noFurtherReceipts)
	}

	d.NeedSpace(7, body)
	d.Row(body, false)
	d.Row(body, false)
	place := strings.TrimSpace(r.Issuer.Place + ", " + r.IssueDate.Format(dateFormat))
	place = strings.TrimPrefix(place, ", ")
	d.Row(body, false, pdfdoc.Cell{Text: place})
	d.Row(body, false)
	d.Row(body, false, pdfdoc.Cell{Text: "_______________________________________________"})
	d.Row(small, false, pdfdoc.Cell{Text: "(Ort, Datum und Unterschrift des Zuwendungsempfängers)"})
	if r.Issuer.Signatory != "" {
		d.Row(body, false, pdfdoc.Cell{Text: r.Issuer.Signatory})
	}
	d.Row(body, false)
	d.NeedSpace(8, small)
	d.Line(small, false, liability)
	d.Line(small, false, recognition)

	if collective {
		r.writeAnnex(d)
	}
}

// writeMarks flags a duplicate or voided receipt.
func (r Receipt) writeMarks(d *pdfdoc.Doc) {
	var marks []string
	if r.Duplicate {
		marks = append(marks, "DUPLIKAT")
	}
	if r.Voided {
		marks = append(marks, "STORNIERT - UNGÜLTIG")
	}
	if len(marks) > 0 {
		d.Row(14, true, pdfdoc.Cell{Text: strings.Join(marks, " / ")})
	}
}

func (r Receipt) writeAnnex(d *pdfdoc.Doc) {
	d.NewPage()
	r.writeMarks(d)
	d.Row(12, true, pdfdoc.Cell{Text: "Anlage zur Sammelbestätigung"})
	d.Row(body, false, pdfdoc.Cell{Text: "Lfd. Nr. " + r.Number + " vom " + r.IssueDate.Format(dateFormat) + " für " + r.RecipientName})
	d.Row(body, false)
	cols := []int{0, 125, 275, 425}
	d.Row(body, true, pdfdoc.Cell{X: cols[0], Text: "Datum der Zuwendung"}, pdfdoc.Cell{X: cols[1], Text: "Art der Zuwendung"},
		pdfdoc.Cell{X: cols[2], Text: "Verzicht auf Erstattung"}, pdfdoc.Cell{X: cols[3], Text: "Betrag"})
	d.Row(small, false, pdfdoc.Cell{X: cols[1], Text: "(Geldzuwendung/Mitgliedsbeitrag)"},
		pdfdoc.Cell{X: cols[2], Text: "von Aufwendungen (ja/nein)"})
	for _, it := range r.Items {
		kind := "Geldzuwendung"
		if it.Kind == ItemMembershipFee {
			kind = "Mitgliedsbeitrag"
		}
		waiver := "nein"
		if it.ExpenseWaiver {
			waiver = "ja"
		}
		d.Row(body, false, pdfdoc.Cell{X: cols[0], Text: it.Date.Format(dateFormat)}, pdfdoc.Cell{X: cols[1], Text: kind},
			pdfdoc.Cell{X: cols[2], Text: waiver}, pdfdoc.Cell{X: cols[3], Text: Amount(it.AmountCents)})
	}
	d.Row(body, false)
	d.Row(body, true, pdfdoc.Cell{X: cols[0], Text: "Gesamtsumme"}, pdfdoc.Cell{X: cols[3], Text: Amount(r.TotalCents())})
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}
