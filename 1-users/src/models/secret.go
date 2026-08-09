package models

import "encoding/json"

const redacted = "[REDACTED]"

// Secret is a string that refuses to print itself.
//
// The underlying kind is string, so inbound JSON decodes exactly as it did
// before; only the outbound directions are redacted. Reading the real value
// takes an explicit string(...) conversion, which keeps every use of a secret
// greppable and visible in review.
//
// Caveat: %#v looks for GoStringer rather than Stringer, so it still dumps the
// raw value. Prefer %v or %s when formatting anything that holds a Secret.
type Secret string

// String satisfies fmt.Stringer, which fmt consults for the %v, %s, %q and %x
// verbs, so an accidental log of a whole struct prints the placeholder.
func (Secret) String() string { return redacted }

// MarshalJSON keeps a Secret out of any response that marshals it, while
// leaving UnmarshalJSON to the default string behaviour so requests still
// carry a real password in.
func (Secret) MarshalJSON() ([]byte, error) { return json.Marshal(redacted) }
