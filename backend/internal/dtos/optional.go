package dtos

import "encoding/json"

// OptionalString distinguishes the three things a field can be in a PATCH
// body: absent, explicitly null, or a value. A plain *string only expresses
// two of them, which forced "" to double as the clear signal.
type OptionalString struct {
	Set   bool
	Value *string
}

// SetString and ClearString build the two "present" cases, mostly for tests.
func SetString(v string) OptionalString { return OptionalString{Set: true, Value: &v} }

func ClearString() OptionalString { return OptionalString{Set: true} }

// UnmarshalJSON runs only when the key is present in the body, which is what
// makes Set meaningful - an absent key leaves the zero value alone.
func (o *OptionalString) UnmarshalJSON(data []byte) error {
	o.Set = true

	if string(data) == "null" {
		o.Value = nil
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	o.Value = &s

	return nil
}
