package sepa

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Test identifiers: example IBANs from bank documentation and the test
// creditor identifier of the Deutsche Bundesbank; none is a real account.
const (
	testCreditorID = "DE98ZZZ09999999999"
	testClubIBAN   = "DE02120300000000202051"
	testIBAN1      = "DE89370400440532013000"
	testIBAN2      = "DE02500105170137075030"
)

func TestValidIBAN(t *testing.T) {
	for _, iban := range []string{testClubIBAN, testIBAN1, testIBAN2, "AT611904300234573201", "FR1420041010050500013M02606"} {
		if !ValidIBAN(iban) {
			t.Errorf("ValidIBAN(%s) = false", iban)
		}
	}
	for _, iban := range []string{"", "DE89370400440532013001", "DE8937040044053201300", "de89370400440532013000", "XX00"} {
		if ValidIBAN(iban) {
			t.Errorf("ValidIBAN(%q) = true", iban)
		}
	}
	// Correct check digits, but one digit too long for a German IBAN.
	if ValidIBAN("DE543704004405320130001") || WellFormedIBAN("DE543704004405320130001") {
		t.Error("23-character German IBAN accepted")
	}
	for _, iban := range []string{"GB82WEST12345698765432", "NL91ABNA0417164300", "BE68539007547034", "NO9386011117947", "CH9300762011623852957"} {
		if !ValidIBAN(iban) {
			t.Errorf("ValidIBAN(%s) = false", iban)
		}
	}
	if !WellFormedIBAN("DE89370400440532013001") || WellFormedIBAN("XX89370400440532013000") {
		t.Error("WellFormedIBAN must ignore check digits and reject unknown countries")
	}
	if got := NormalizeIBAN(" de89 3704 0044 0532 0130 00 "); got != testIBAN1 {
		t.Errorf("NormalizeIBAN = %s", got)
	}
}

func TestValidCreditorID(t *testing.T) {
	for _, id := range []string{testCreditorID, "DE98ABC09999999999"} {
		if !ValidCreditorID(id) {
			t.Errorf("ValidCreditorID(%s) = false", id)
		}
	}
	for _, id := range []string{"", "DE97ZZZ09999999999", "DE98ZZZ", "DE98ZZZ0999999999!"} {
		if ValidCreditorID(id) {
			t.Errorf("ValidCreditorID(%q) = true", id)
		}
	}
}

func TestValidBICAndMandate(t *testing.T) {
	for _, bic := range []string{"COBADEFFXXX", "COBADEFF", "INGDDEFF"} {
		if !ValidBIC(bic) {
			t.Errorf("ValidBIC(%s) = false", bic)
		}
	}
	for _, bic := range []string{"", "COBADE", "cobadeff", "COBADEFF12"} {
		if ValidBIC(bic) {
			t.Errorf("ValidBIC(%q) = true", bic)
		}
	}
	if !ValidMandateReference("NW-1001") || ValidMandateReference("NW 1001") || ValidMandateReference(strings.Repeat("A", 36)) {
		t.Error("ValidMandateReference")
	}
}

func TestText(t *testing.T) {
	if got := Text("Jahresbeitrag 2026 – Greta Möller & Söhne_€", 140); got != "Jahresbeitrag 2026 - Greta Moeller + Soehne-" {
		t.Errorf("Text = %q", got)
	}
	if got := Text("Straße", 4); got != "Stra" {
		t.Errorf("Text cut = %q", got)
	}
}

func testInitiation() Initiation {
	mandate := time.Date(2022, 4, 23, 0, 0, 0, 0, time.UTC)
	return Initiation{
		MessageID:      "0192f1c0a4b37c3e9d1f2a3b4c5d6e7f",
		Created:        time.Date(2026, 11, 1, 9, 30, 0, 0, time.UTC),
		CollectionDate: time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC),
		Creditor:       Creditor{Name: "Kiteclub Nordwind e.V.", CreditorID: testCreditorID, IBAN: testClubIBAN, BIC: "BYLADEM1001"},
		Debits: []Debit{
			{EndToEndID: "a1", AmountCents: 18000, SequenceType: SeqFirst, MandateID: "NW-1001", MandateDate: mandate,
				DebtorName: "Jannik Petersen", IBAN: testIBAN1, BIC: "COBADEFFXXX", Remittance: "Jahresbeitrag 2026 Mitglied 1001"},
			{EndToEndID: "a2", AmountCents: 9050, SequenceType: SeqRecurring, MandateID: "NW-1008", MandateDate: mandate,
				DebtorName: "Kira Thomsen", IBAN: testIBAN2, Remittance: "Jahresbeitrag 2026 Mitglied 1008"},
			{EndToEndID: "a3", AmountCents: 5, SequenceType: SeqRecurring, MandateID: "NW-1015", MandateDate: mandate,
				DebtorName: "Greta Möller", IBAN: testIBAN1},
		},
	}
}

func TestXMLStructure(t *testing.T) {
	out, err := testInitiation().XML()
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{
		`<Document xmlns="urn:iso:std:iso:20022:tech:xsd:pain.008.001.08">`,
		`<NbOfTxs>3</NbOfTxs>`, `<CtrlSum>270.55</CtrlSum>`, `<CtrlSum>90.55</CtrlSum>`,
		`<SeqTp>FRST</SeqTp>`, `<SeqTp>RCUR</SeqTp>`, `<ReqdColltnDt>2026-11-15</ReqdColltnDt>`,
		`<InstdAmt Ccy="EUR">0.05</InstdAmt>`, `<Id>NOTPROVIDED</Id>`, `<Nm>Greta Moeller</Nm>`,
		`<Id>DE98ZZZ09999999999</Id>`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %s in\n%s", want, s)
		}
	}
	if strings.Count(s, "<PmtInf>") != 2 {
		t.Errorf("want 2 payment infos")
	}
	if _, err := (Initiation{}).XML(); err == nil {
		t.Error("empty initiation must fail")
	}
}

// TestXMLValidatesAgainstXSD checks the generated file against the official
// ISO 20022 schema (testdata/pain.008.001.08.xsd). It needs python3 with
// lxml, as Go has no XSD validator; without it the test is skipped.
func TestXMLValidatesAgainstXSD(t *testing.T) {
	python, err := exec.LookPath("python3")
	if err != nil || exec.Command(python, "-c", "import lxml").Run() != nil {
		t.Skip("python3 with lxml not available")
	}
	out, err := testInitiation().XML()
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "pain008.xml")
	if err := os.WriteFile(file, out, 0o600); err != nil {
		t.Fatal(err)
	}
	const script = `import sys
from lxml import etree
schema = etree.XMLSchema(etree.parse(sys.argv[1]))
doc = etree.parse(sys.argv[2])
if not schema.validate(doc):
    print(schema.error_log)
    sys.exit(1)`
	res, err := exec.Command(python, "-I", "-c", script, filepath.Join("testdata", "pain.008.001.08.xsd"), file).CombinedOutput()
	if err != nil {
		t.Fatalf("XSD validation failed: %v\n%s\n%s", err, res, out)
	}
}
