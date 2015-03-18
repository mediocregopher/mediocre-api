package resphelper

import (
	. "testing"

	"github.com/mediocregopher/radix.v2/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnmarshalResp(t *T) {
	type Outer struct {
		A int64
		B int
		C []byte
		D string
		E int `resp:"ee"`
	}

	f := Outer{}
	r := redis.NewResp(map[string]interface{}{
		"A":  5,
		"B":  6,
		"C":  []byte("hello"),
		"D":  "world",
		"ee": 7,
	})
	require.Nil(t, UnmarshalResp(r, &f))
	assert.Equal(t, 5, f.A)
	assert.Equal(t, 6, f.B)
	assert.Equal(t, []byte("hello"), f.C)
	assert.Equal(t, "world", f.D)
	assert.Equal(t, 7, f.E)
}

func TestUnmarshalRespInner(t *T) {
	type InnerA struct {
		Foo string
	}
	type InnerB struct {
		Bar string
	}
	type InnerC struct {
		Baz string
	}
	type Outer struct {
		A int64
		B InnerA
		C *InnerB
		D *InnerC
		E string // not going to be found
	}
	f := Outer{D: &InnerC{}}
	r := redis.NewResp(map[string]interface{}{
		"A": 5,
		"B": map[string]interface{}{
			"Foo": "hello",
		},
		"C": map[string]interface{}{
			"Bar": "world",
		},
		"D": map[string]interface{}{
			"Baz": "again",
		},
	})
	require.Nil(t, UnmarshalResp(r, &f))
	assert.Equal(t, 5, f.A)
	assert.Equal(t, "hello", f.B.Foo)
	assert.Equal(t, "world", f.C.Bar)
	assert.Equal(t, "again", f.D.Baz)
	assert.Equal(t, "", f.E)

	f = Outer{D: &InnerC{}}
	r = redis.NewResp(map[string]interface{}{
		"A":   5,
		"Foo": "hello",
		"Bar": "world",
		"Baz": "again",
	})
	require.Nil(t, UnmarshalResp(r, &f))
	assert.Equal(t, 5, f.A)
	assert.Equal(t, "hello", f.B.Foo)
	assert.Equal(t, "world", f.C.Bar)
	assert.Equal(t, "again", f.D.Baz)
	assert.Equal(t, "", f.E)
}
