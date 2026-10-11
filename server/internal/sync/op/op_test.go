package op

import (
	"encoding/json"
	"testing"
)

func TestOpJSONFormat(t *testing.T) {
	o := Op{
		T:   "members",
		PK:  map[string]any{"id": "0190f5d0"},
		HLC: "0000019a1b2c-0003",
		Set: map[string]any{"number": int64(1042), "last_name": "Mustermann", "comment": nil},
	}
	b, err := json.Marshal(o)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"t":"members","pk":{"id":"0190f5d0"},"hlc":"0000019a1b2c-0003","set":{"comment":null,"last_name":"Mustermann","number":1042}}`
	if string(b) != want {
		t.Errorf("got  %s\nwant %s", b, want)
	}
}

func TestPKAndValueRoundTrip(t *testing.T) {
	pk := map[string]any{"role_id": "r", "member_id": "m"}
	text := EncodePK(pk)
	if text != `{"member_id":"m","role_id":"r"}` {
		t.Errorf("EncodePK = %s, want sorted keys", text)
	}
	back, err := DecodePK(text)
	if err != nil || back["member_id"] != "m" || back["role_id"] != "r" {
		t.Errorf("DecodePK = %v, %v", back, err)
	}

	for _, v := range []any{int64(7), 7, "x@example.org", nil, 1.5, true} {
		got, err := DecodeValue(EncodeValue(v))
		if err != nil || got != Normalize(v) {
			t.Errorf("round trip %v (%T) = %v (%T), %v", v, v, got, got, err)
		}
	}
	if Normalize([]byte("a")) != "a" || Normalize(int32(3)) != int64(3) {
		t.Error("Normalize")
	}
}
