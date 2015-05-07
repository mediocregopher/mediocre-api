// Package pickyjson is useful for extracting simple json types with certain
// constraints. For example, extracting a string but only of a certain type, or
// with a default value if not set
//
// TODO example
package pickyjson

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/mediocregopher/mediocre-api/common"
)

// Various errors which may be returned by this package
var (
	ErrTooLong   = common.ExpectedErr{Code: 400, Err: "too long"}
	ErrTooShort  = common.ExpectedErr{Code: 400, Err: "too short"}
	ErrMalformed = common.ExpectedErr{Code: 400, Err: "malformed"}
	ErrTooBig    = common.ExpectedErr{Code: 400, Err: "too big"}
	ErrTooSmall  = common.ExpectedErr{Code: 400, Err: "too small"}
)

// Functions which return errors based on the related field names
var (
	ErrFieldRequiredf = func(f string) error {
		return common.ExpectedErrf(400, "field %s required", f)
	}
)

// Str is a wrapper for a normal go string, but with extra constraints. If a
// constraint is not specified it will not be applied
type Str struct {
	// Maximum and minimum lengths that the string may be. MinLength can be used
	// essentially require the Str to be set if it's a field in a struct
	MaxLength, MinLength int

	// A function the string will be passed to, useful for more complicated
	// checks. It returns whether or not the string is valid
	Func func(string) bool

	// A function which can spit out a new form of the string value (assuming it
	// passes all other constraints)
	Map func(string) (string, error)

	// The place the value will be filled into if it passes all constraints.
	// This can be pre-filled with a default value
	Str string
}

// MarshalJSON implements the json.Marshaler interface, marshalling the value of
// the Str field
func (s *Str) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Str)
}

// UnmarshalJSON implements the json.Unmarshaler interface, unmarshalling the
// given encoded json into the Str field. If the value doesn't fit within any
// of the constraints an error will be returned
func (s *Str) UnmarshalJSON(b []byte) error {
	var err error
	if err = json.Unmarshal(b, &s.Str); err != nil {
		return err
	}

	if l := len(s.Str); s.MaxLength > 0 && l > s.MaxLength {
		return ErrTooLong
	} else if l < s.MinLength {
		return ErrTooShort
	}

	if s.Str != "" && s.Func != nil && !s.Func(s.Str) {
		return ErrMalformed
	}

	if s.Str != "" && s.Map != nil {
		if s.Str, err = s.Map(s.Str); err != nil {
			return err
		}
	}

	return nil
}

// String implementation for fmt.Stringer
func (s *Str) String() string {
	return fmt.Sprintf("%q", s.Str)
}

// Required is a convenience method which returns an exact copy of the String
// being called on except with a MinLength of 1 (if MinLength wasn't already
// set)
func (s Str) Required() Str {
	if s.MinLength < 1 {
		s.MinLength = 1
	}
	return s
}

// Int64 is a wrapper for a normal go int64, but with extra constraints. If a
// constraint is not specified it will not be applied
type Int64 struct {
	// Maximum and minimum values that the integer may be
	//
	// By default (both 0) this will only allow for positive integers. Min can
	// be set to math.MinInt64 to allow for negative integers. If one is set the
	// other is assumed to be set as well
	Max, Min int64

	// A function the integer will be passed to. It returns whether or not the
	// integer is valid
	Func func(int64) bool

	// The place the value will be filled into if it passes all constraints.
	// This can be pre-filled with a default value
	Int64 int64

	// Whether or not this must be filled in, if specified for a field in a
	// struct
	Require bool

	filled bool
}

// MarshalJSON implements the json.Marshaler interface, marshalling the value of
// the Int64 field
func (i *Int64) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.Int64)
}

// UnmarshalJSON implements the json.Unmarshaler interface, unmarshalling the
// given encoded json into the Int64 field. If the value doesn't fit within any
// of the constraints an error will be returned
func (i *Int64) UnmarshalJSON(b []byte) error {
	if err := json.Unmarshal(b, &i.Int64); err != nil {
		return err
	}

	if i.Max > i.Min && i.Int64 > i.Max {
		return ErrTooBig
	} else if i.Int64 < i.Min {
		return ErrTooSmall
	}

	if i.Func != nil && !i.Func(i.Int64) {
		return ErrMalformed
	}

	i.filled = true

	return nil
}

// Required is a convenience method which returns an exact copy of the Int64
// with Require set to true
func (i Int64) Required() Int64 {
	i.Require = true
	return i
}

// String implementation for fmt.Stringer
func (i *Int64) String() string {
	return fmt.Sprint(i.Int64)
}

// CheckRequired takes in a struct and looks through it to ensure all required
// parameters were actually filled in post-unmarshal. It will look through all
// struct recursively (although it won't traverse slices/maps at the moment)
func CheckRequired(i interface{}) error {
	v := reflect.ValueOf(i)
	vk := v.Kind()
	if vk == reflect.Ptr || vk == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}
	t := v.Type()

	for ii := 0; ii < v.NumField(); ii++ {
		fieldValV := v.Field(ii)
		switch fieldVal := fieldValV.Interface().(type) {
		case Str:
			if fieldVal.MinLength > 0 && fieldVal.Str == "" {
				field := t.Field(ii)
				return ErrFieldRequiredf(field.Name)
			}
		case Int64:
			if fieldVal.Require && !fieldVal.filled {
				field := t.Field(ii)
				return ErrFieldRequiredf(field.Name)
			}
		default:
			fvk := fieldValV.Kind()
			if fvk == reflect.Ptr || fvk == reflect.Interface {
				fieldValV = fieldValV.Elem()
			}
			if fieldValV.Kind() == reflect.Struct {
				if err := CheckRequired(fieldVal); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
