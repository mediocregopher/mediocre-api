package pickyjson

import (
	"encoding/json"
	"strings"
	. "testing"

	"github.com/stretchr/testify/require"
)

func unmarshal(s string, i interface{}) error {
	return json.Unmarshal([]byte(s), i)
}

func TestStr(t *T) {
	s := Str{
		MaxLength: 5,
		MinLength: 2,
	}
	require.Equal(t, ErrTooShort, unmarshal(`"f"`, &s))
	require.Equal(t, ErrTooLong, unmarshal(`"foobar"`, &s))
	require.Nil(t, unmarshal(`"foo"`, &s))
	require.Equal(t, "foo", s.Str)

	s = Str{
		Func: func(str string) bool {
			return str[0] == 'f'
		},
	}
	require.Equal(t, ErrMalformed, unmarshal(`"bar"`, &s))
	require.Nil(t, unmarshal(`"foo"`, &s))
	require.Equal(t, "foo", s.Str)

	s = Str{
		Map: func(str string) (string, error) {
			if str[0] == 'f' {
				return "", ErrMalformed // as an example
			}
			return strings.ToUpper(str), nil
		},
	}
	require.Equal(t, ErrMalformed, unmarshal(`"foo"`, &s))
	require.Nil(t, unmarshal(`"bar"`, &s))
	require.Equal(t, "BAR", s.Str)
}

func TestInt64(t *T) {
	i := Int64{}
	require.Equal(t, ErrTooSmall, unmarshal(`-1`, &i))
	require.Nil(t, unmarshal(`4`, &i))
	require.Equal(t, int64(4), i.Int64)

	i = Int64{
		Max: 10,
		Min: 2,
	}
	require.Equal(t, ErrTooSmall, unmarshal(`1`, &i))
	require.Equal(t, ErrTooBig, unmarshal(`11`, &i))
	require.Nil(t, unmarshal(`4`, &i))
	require.Equal(t, int64(4), i.Int64)

	i = Int64{
		Min: -5,
	}
	require.Equal(t, ErrTooSmall, unmarshal(`-6`, &i))
	require.Equal(t, ErrTooBig, unmarshal(`1`, &i))
	require.Nil(t, unmarshal(`0`, &i))
	require.Equal(t, int64(0), i.Int64)
	require.Nil(t, unmarshal(`-5`, &i))
	require.Equal(t, int64(-5), i.Int64)

	i = Int64{
		Func: func(ii int64) bool {
			return ii%2 == 0
		},
	}
	require.Equal(t, ErrMalformed, unmarshal(`1`, &i))
	require.Nil(t, unmarshal(`2`, &i))
	require.Equal(t, int64(2), i.Int64)
}

func TestCheckRequired(t *T) {
	type J struct {
		S1, S2 Str
		I1, I2 Int64
		J2     struct {
			S3, S4 Str
		}
	}

	j := J{
		S1: Str{},
		S2: Str{}.Required(),
		I1: Int64{},
		I2: Int64{}.Required(),
		J2: struct{ S3, S4 Str }{
			S3: Str{},
			S4: Str{}.Required(),
		},
	}

	err := CheckRequired(&j)
	require.Equal(t, "field S2 required", err.Error())

	// S1 isn't required, shouldn't affect anything
	j.S1.Str = "Foo"
	err = CheckRequired(&j)
	require.Equal(t, "field S2 required", err.Error())

	// S2, I2, and S4 still required

	j.S2.Str = "Bar"
	err = CheckRequired(&j)
	require.Equal(t, "field I2 required", err.Error())

	// I2 and S4 still required

	j.I1.Int64 = 1
	j.I1.filled = true
	err = CheckRequired(&j)
	require.Equal(t, "field I2 required", err.Error())

	// I2 and S4 still required

	j.I2.Int64 = 1
	j.I2.filled = true
	err = CheckRequired(&j)
	require.Equal(t, "field S4 required", err.Error())

	// S4 still required

	j.J2.S3.Str = "Baz"
	err = CheckRequired(&j)
	require.Equal(t, "field S4 required", err.Error())

	// S4 still required

	j.J2.S4.Str = "Buz"
	err = CheckRequired(&j)
	require.Nil(t, err)
}
