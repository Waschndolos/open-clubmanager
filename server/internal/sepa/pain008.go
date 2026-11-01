package sepa

import (
	"encoding/xml"
	"errors"
	"fmt"
	"time"
)

// Sequence types of a SEPA Core direct debit: the first collection under a
// mandate and every later one.
const (
	SeqFirst     = "FRST"
	SeqRecurring = "RCUR"
)

// Namespace is the XML namespace of pain.008.001.08.
const Namespace = "urn:iso:std:iso:20022:tech:xsd:pain.008.001.08"

// Creditor is the club collecting the debits.
type Creditor struct {
	Name       string
	CreditorID string
	IBAN       string
	BIC        string // optional (IBAN-only)
}

// Debit is one collection from a debtor's account.
type Debit struct {
	EndToEndID   string
	AmountCents  int64
	SequenceType string
	MandateID    string
	MandateDate  time.Time
	DebtorName   string
	IBAN         string
	BIC          string // optional (IBAN-only)
	Remittance   string
}

// Initiation is a direct debit file: all debits are collected on the same
// date, grouped into one payment information block per sequence type.
type Initiation struct {
	MessageID      string
	Created        time.Time
	CollectionDate time.Time
	Creditor       Creditor
	Debits         []Debit
}

type document struct {
	XMLName xml.Name `xml:"Document"`
	Xmlns   string   `xml:"xmlns,attr"`
	Init    struct {
		GrpHdr struct {
			MsgID    string `xml:"MsgId"`
			CreDtTm  string `xml:"CreDtTm"`
			NbOfTxs  int    `xml:"NbOfTxs"`
			CtrlSum  string `xml:"CtrlSum"`
			InitgPty party  `xml:"InitgPty"`
		} `xml:"GrpHdr"`
		PmtInf []paymentInfo `xml:"PmtInf"`
	} `xml:"CstmrDrctDbtInitn"`
}

type party struct {
	Nm string `xml:"Nm"`
}

type account struct {
	IBAN string `xml:"Id>IBAN"`
}

// agent is a bank, identified by BIC or, for IBAN-only payments, as
// "NOTPROVIDED".
type agent struct {
	BICFI string   `xml:"FinInstnId>BICFI,omitempty"`
	Othr  *otherID `xml:"FinInstnId>Othr,omitempty"`
}

type otherID struct {
	ID string `xml:"Id"`
}

func newAgent(bic string) agent {
	if bic == "" {
		return agent{Othr: &otherID{ID: "NOTPROVIDED"}}
	}
	return agent{BICFI: bic}
}

type remittance struct {
	Ustrd string `xml:"Ustrd"`
}

type paymentInfo struct {
	PmtInfID  string        `xml:"PmtInfId"`
	PmtMtd    string        `xml:"PmtMtd"`
	BtchBookg bool          `xml:"BtchBookg"`
	NbOfTxs   int           `xml:"NbOfTxs"`
	CtrlSum   string        `xml:"CtrlSum"`
	SvcLvl    string        `xml:"PmtTpInf>SvcLvl>Cd"`
	LclInstrm string        `xml:"PmtTpInf>LclInstrm>Cd"`
	SeqTp     string        `xml:"PmtTpInf>SeqTp"`
	ReqdColtn string        `xml:"ReqdColltnDt"`
	Cdtr      party         `xml:"Cdtr"`
	CdtrAcct  account       `xml:"CdtrAcct"`
	CdtrAgt   agent         `xml:"CdtrAgt"`
	ChrgBr    string        `xml:"ChrgBr"`
	SchmeID   string        `xml:"CdtrSchmeId>Id>PrvtId>Othr>Id"`
	SchmeNm   string        `xml:"CdtrSchmeId>Id>PrvtId>Othr>SchmeNm>Prtry"`
	Tx        []transaction `xml:"DrctDbtTxInf"`
}

type transaction struct {
	EndToEndID string `xml:"PmtId>EndToEndId"`
	Amount     struct {
		Ccy   string `xml:"Ccy,attr"`
		Value string `xml:",chardata"`
	} `xml:"InstdAmt"`
	MndtID    string      `xml:"DrctDbtTx>MndtRltdInf>MndtId"`
	DtOfSgntr string      `xml:"DrctDbtTx>MndtRltdInf>DtOfSgntr"`
	DbtrAgt   agent       `xml:"DbtrAgt"`
	Dbtr      party       `xml:"Dbtr"`
	DbtrAcct  account     `xml:"DbtrAcct"`
	RmtInf    *remittance `xml:"RmtInf,omitempty"`
}

// Amount formats cents as a decimal amount with two fraction digits.
func Amount(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

// XML renders the initiation as a pain.008.001.08 document. The caller has
// validated the identifiers; names and remittance texts are converted to the
// SEPA character set here.
func (in Initiation) XML() ([]byte, error) {
	if len(in.Debits) == 0 {
		return nil, errors.New("no debits")
	}
	var doc document
	doc.Xmlns = Namespace
	hdr := &doc.Init.GrpHdr
	hdr.MsgID = in.MessageID
	hdr.CreDtTm = in.Created.UTC().Format("2006-01-02T15:04:05")
	hdr.InitgPty = party{Nm: Text(in.Creditor.Name, 70)}

	var total int64
	for _, seq := range []string{SeqFirst, SeqRecurring} {
		info := paymentInfo{
			PmtInfID: Text(in.MessageID, 30) + "-" + seq, PmtMtd: "DD", BtchBookg: true,
			SvcLvl: "SEPA", LclInstrm: "CORE", SeqTp: seq, ReqdColtn: in.CollectionDate.Format("2006-01-02"),
			Cdtr: party{Nm: Text(in.Creditor.Name, 70)}, CdtrAcct: account{IBAN: in.Creditor.IBAN},
			CdtrAgt: newAgent(in.Creditor.BIC), ChrgBr: "SLEV", SchmeID: in.Creditor.CreditorID, SchmeNm: "SEPA",
		}
		var sum int64
		for _, d := range in.Debits {
			if d.SequenceType != seq {
				continue
			}
			if d.AmountCents <= 0 {
				return nil, fmt.Errorf("debit %s: amount must be positive", d.EndToEndID)
			}
			tx := transaction{
				EndToEndID: d.EndToEndID, MndtID: d.MandateID, DtOfSgntr: d.MandateDate.Format("2006-01-02"),
				DbtrAgt: newAgent(d.BIC), Dbtr: party{Nm: Text(d.DebtorName, 70)}, DbtrAcct: account{IBAN: d.IBAN},
			}
			if r := Text(d.Remittance, 140); r != "" {
				tx.RmtInf = &remittance{Ustrd: r}
			}
			tx.Amount.Ccy, tx.Amount.Value = "EUR", Amount(d.AmountCents)
			info.Tx = append(info.Tx, tx)
			sum += d.AmountCents
		}
		if len(info.Tx) == 0 {
			continue
		}
		info.NbOfTxs, info.CtrlSum = len(info.Tx), Amount(sum)
		doc.Init.PmtInf = append(doc.Init.PmtInf, info)
		hdr.NbOfTxs += len(info.Tx)
		total += sum
	}
	if hdr.NbOfTxs != len(in.Debits) {
		return nil, errors.New("debit with unknown sequence type")
	}
	hdr.CtrlSum = Amount(total)

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), append(out, '\n')...), nil
}
