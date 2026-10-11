package layout

import (
	"testing"

	"github.com/Waschndolos/open-clubmanager/server/internal/db"
)

const dev = "abcdefghijklmnopqrstuvwxyz"

func TestParse(t *testing.T) {
	valid := map[string]Entry{
		"club.json":                                            {Kind: KindGenesis},
		"club.json.sig":                                        {Kind: KindGenesisSig},
		"acl/00000007-0123456789abcdef.json":                   {Kind: KindACL, Version: 7, HashPrefix: "0123456789abcdef"},
		"acl/00000007-0123456789abcdef.json.sig":               {Kind: KindACLSig, Version: 7, HashPrefix: "0123456789abcdef"},
		"keys/members.core/2/" + dev + ".age":                  {Kind: KindKey, Scope: db.ScopeMembersCore, Epoch: 2, Recipient: dev},
		"keys/finance/12/root.age":                             {Kind: KindKey, Scope: db.ScopeFinance, Epoch: 12, Recipient: "root"},
		"requests/" + dev + ".json":                            {Kind: KindRequest, DeviceID: dev},
		"requests/" + dev + ".json.sig":                        {Kind: KindRequestSig, DeviceID: dev},
		"segments/" + dev + "/inventory/000000000001.seg":      {Kind: KindSegment, DeviceID: dev, Scope: db.ScopeInventory, Seq: 1},
		"snapshots/admin/1/0000019a1b2c-0003-" + dev + ".snap": {Kind: KindSnapshot, Scope: db.ScopeAdmin, Epoch: 1, HLC: "0000019a1b2c-0003", DeviceID: dev},
	}
	for p, want := range valid {
		got, ok := Parse(p)
		if !ok || got != want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", p, got, ok, want)
		}
	}

	invalid := []string{
		// Temp files and sync-client conflict copies (§6).
		"acl/.tmp-0123abcd",
		".tmp-0123abcd",
		"segments/" + dev + "/inventory/.tmp-1a2b",
		"segments/" + dev + "/inventory/000000000001 (conflicted copy).seg",
		"segments/" + dev + "/inventory/000000000001-Konflikt.seg",
		"segments/" + dev + "/inventory/000000000001 (Konflikt 2026-10-11).seg",
		"segments/" + dev + "/inventory/000000000001.seg.tmp",
		"segments/" + dev + "/inventory/000000000001 (1).seg",
		"club (conflicted copy).json",
		"club.json~",
		"Club.json",
		"acl/00000007-0123456789abcdef (Manu's conflicted copy 2026-10-11).json",
		"acl/00000007-0123456789abcdef-1.json",
		"keys/finance/1/" + dev + ".age.conflict",
		"requests/" + dev + " (1).json",
		// Wrong shapes.
		"acl/7-0123456789abcdef.json",
		"acl/00000000-0123456789abcdef.json",
		"acl/00000007-0123456789ABCDEF.json",
		"acl/00000007-0123456789abcde.json",
		"keys/finance/0/root.age",
		"keys/finance/01/root.age",
		"keys/photos/1/root.age",
		"keys/finance/1/ROOT.age",
		"keys/finance/1/abc.age",
		"requests/" + dev[:25] + ".json",
		"requests/" + dev + "1.json",
		"segments/" + dev + "/inventory/000000000000.seg",
		"segments/" + dev + "/inventory/1.seg",
		"segments/" + dev + "/photos/000000000001.seg",
		"segments/" + dev + "/inventory/0000000000001.seg",
		"segments/" + dev + "/inventory/sub/000000000001.seg",
		"segments/ABCDEFGHIJKLMNOPQRSTUVWXYZ/inventory/000000000001.seg",
		"snapshots/admin/1/0000019a1b2c-0003.snap",
		"/club.json",
		"./club.json",
		"",
	}
	for _, p := range invalid {
		if e, ok := Parse(p); ok {
			t.Errorf("Parse(%q) accepted as %+v", p, e)
		}
	}
}

func TestPathsRoundTrip(t *testing.T) {
	var sum [32]byte
	sum[0], sum[7] = 0xab, 0x01
	cases := map[string]Entry{
		ACLPath(3, sum):                                            {Kind: KindACL, Version: 3, HashPrefix: "ab00000000000001"},
		SigPath(ACLPath(3, sum)):                                   {Kind: KindACLSig, Version: 3, HashPrefix: "ab00000000000001"},
		KeyPath(db.ScopeMembersBank, 4, dev):                       {Kind: KindKey, Scope: db.ScopeMembersBank, Epoch: 4, Recipient: dev},
		KeyPath(db.ScopeMembersBank, 4, RootRecipient):             {Kind: KindKey, Scope: db.ScopeMembersBank, Epoch: 4, Recipient: "root"},
		RequestPath(dev):                                           {Kind: KindRequest, DeviceID: dev},
		SegmentPath(dev, db.ScopeFinance, 42):                      {Kind: KindSegment, DeviceID: dev, Scope: db.ScopeFinance, Seq: 42},
		SnapshotPath(db.ScopeFinance, 2, "0000019a1b2c-0003", dev): {Kind: KindSnapshot, Scope: db.ScopeFinance, Epoch: 2, HLC: "0000019a1b2c-0003", DeviceID: dev},
	}
	for p, want := range cases {
		if got, ok := Parse(p); !ok || got != want {
			t.Errorf("Parse(%q) = %+v, %v; want %+v", p, got, ok, want)
		}
	}
	if SegmentPath(dev, db.ScopeFinance, 1) != "segments/"+dev+"/finance/000000000001.seg" {
		t.Error("segment path not zero-padded to 12 digits")
	}
	if ACLPath(7, sum) != "acl/00000007-ab00000000000001.json" {
		t.Error("ACL path not zero-padded to 8 digits")
	}
}
