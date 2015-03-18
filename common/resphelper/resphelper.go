// Package resphelper provides a number of methods for working with radix.v2
// Resp structs
package resphelper

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/mediocregopher/radix.v2/redis"
)

var bytesType = reflect.TypeOf([]byte(nil))

// UnmarshalResp takes a *redis.Resp and attemps to unmarshal its data into the
// given struct or pointer to a struct.
//
// Struct fields must be of types string, []byte, int, or int64. Embedded
// structs and pointers to structs are also allowed
//
// A struct field can tag itself using "resp" in order to change the name of the
// key it will look at when filling itself in.
//
//	type Foo struct {
//		A string
//		B int `resp: "bb"`
//	}
//	r := conn.Cmd("HGETALL", "foo") // retrieve resp
//	f := Foo{}
//	if err := resphelper.UnmarshalResp(r, &f); err != nil {
//		// handle error
//	}
//
func UnmarshalResp(r *redis.Resp, i interface{}) error {
	v, err := getStructPtrValue(i)
	if err != nil {
		return err
	}
	t := v.Type()

	rm, err := respToMap(r)
	if err != nil {
		return err
	}

	for ii, n := 0, t.NumField(); ii < n; ii++ {
		structField := t.Field(ii)
		fieldV := v.Field(ii)
		fieldT := fieldV.Type()
		fieldName := structField.Name
		if tagName := structField.Tag.Get("resp"); tagName != "" {
			fieldName = tagName
		}

		rv, ok := rm[fieldName]

		var fieldVV interface{}
		shouldAssign := true
		err = nil
		if fieldK := fieldV.Kind(); fieldK == reflect.Int64 && ok {
			fieldVV, err = rv.Int64()
		} else if fieldK == reflect.Int {
			if !ok {
				continue
			}
			fieldVV, err = rv.Int()
		} else if fieldK == reflect.String {
			if !ok {
				continue
			}
			fieldVV, err = rv.Str()
		} else if fieldT == bytesType {
			if !ok {
				continue
			}
			fieldVV, err = rv.Bytes()
		} else if fieldK == reflect.Struct ||
			(fieldK == reflect.Ptr && fieldT.Elem().Kind() == reflect.Struct) {

			if fieldK == reflect.Struct {
				fieldVV = fieldV.Addr().Interface()
			} else {
				if fieldV.IsNil() {
					fieldV.Set(reflect.New(fieldT.Elem()))
				}
				fieldVV = fieldV.Interface()
			}
			if rv == nil {
				err = UnmarshalResp(r, fieldVV)
			} else if rv.IsType(redis.Array) {
				err = UnmarshalResp(rv, fieldVV)
			} else {
				continue
			}
			shouldAssign = false

		} else {
			err = fmt.Errorf("unsupported UnmarshalResp type %s", fieldK)
		}
		if err != nil {
			return err
		}
		if shouldAssign {
			fieldV.Set(reflect.ValueOf(fieldVV))
		}
	}

	return nil
}

func respToMap(r *redis.Resp) (map[string]*redis.Resp, error) {
	rm := map[string]*redis.Resp{}
	l, err := r.Array()
	if err != nil {
		return nil, err
	}
	if len(l)%2 != 0 {
		return nil, errors.New("resp list doesn't have even number of elements")
	}

	for i := 0; i < len(l); i += 2 {
		key, err := l[i].Str()
		if err != nil {
			return nil, err
		}
		rm[key] = l[i+1]
	}

	return rm, nil
}

func getStructPtrValue(i interface{}) (*reflect.Value, error) {
	v := reflect.ValueOf(i)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		if v.Kind() != reflect.Struct {
			return nil, errors.New("Must give a struct or struct pointer")
		}
	} else {
		return nil, errors.New("Must give a struct or struct pointer")
	}
	return &v, nil
}
