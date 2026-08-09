package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const pw = "DemoPass123!"

// The verbs fmt routes through Stringer. %d and friends are not meaningful for
// a string-kinded type, and %#v is covered separately below.
var safeVerbs = []string{"%v", "%s", "%q", "%x", "%+v"}

// %x hex-encodes everything it touches, so it is useless for asserting that
// plain fields stay readable, even though it still redacts correctly.
var textVerbs = []string{"%v", "%s", "%q", "%+v"}

func TestSecretNeverFormatsItsValue(t *testing.T) {
	s := Secret(pw)
	for _, verb := range safeVerbs {
		if got := fmt.Sprintf(verb, s); strings.Contains(got, pw) {
			t.Errorf("%s leaked the value: %s", verb, got)
		}
	}
}

func TestSecretStaysRedactedInsideAStruct(t *testing.T) {
	u := User{Email: "demo@omni.dev", UnHashedPassword: Secret(pw)}
	for _, verb := range safeVerbs {
		if got := fmt.Sprintf(verb, u); strings.Contains(got, pw) {
			t.Errorf("%s leaked the value from within User: %s", verb, got)
		}
	}
	for _, verb := range textVerbs {
		if got := fmt.Sprintf(verb, u); !strings.Contains(got, "demo@omni.dev") {
			t.Errorf("%s should still print the non-secret fields: %s", verb, got)
		}
	}
}

func TestSecretDoesNotMarshal(t *testing.T) {
	out, err := json.Marshal(User{Email: "demo@omni.dev", UnHashedPassword: Secret(pw)})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(out), pw) {
		t.Errorf("marshalled JSON leaked the value: %s", out)
	}
}

// Inbound decoding has to keep working: login and register both unmarshal a
// request body straight into User and then compare the password.
func TestSecretStillUnmarshals(t *testing.T) {
	var u User
	if err := json.Unmarshal([]byte(`{"email":"demo@omni.dev","password":"`+pw+`"}`), &u); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if string(u.UnHashedPassword) != pw {
		t.Errorf("want %q through an explicit conversion, got %q", pw, string(u.UnHashedPassword))
	}
}

// Documents a known hole rather than asserting a guarantee: %#v looks for
// GoStringer, not Stringer, so it dumps the raw value. If this ever starts
// failing, Secret grew a GoString method and the caveat can come out of the
// doc comment and #21.
func TestSharpVIsStillAKnownLeak(t *testing.T) {
	if !strings.Contains(fmt.Sprintf("%#v", Secret(pw)), pw) {
		t.Skip("sharp-v no longer leaks: update the Secret doc comment and #21")
	}
}
