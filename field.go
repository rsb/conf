package conf

import (
	"encoding"
	"fmt"
	"reflect"

	"github.com/rsb/failure"
)

var (
	InvalidSpecFailure = failure.Config("specification must be a struct pointer")
)

// Field holds information about the current configuration variable
type Field struct {
	Prefix       string
	Name         string
	EnvName      string
	ReflectValue reflect.Value
	ReflectTag   reflect.StructTag
	Tag          Tag
}

func (f Field) EnvKey() string {
	key := f.EnvName
	if f.Prefix != "" {
		key = fmt.Sprintf("%s_%s", f.Prefix, key)
	}

	return key
}

func (f Field) IsRequired() bool {
	return f.Tag.Required
}

func (f Field) IsDefault() bool {
	return f.Tag.IsDefault
}

func (f Field) DefaultValue() string {
	return f.Tag.Default
}

func Fields(spec interface{}, prefixParam ...string) ([]Field, error) {
	var prefix string
	s := reflect.ValueOf(spec)

	if s.Kind() != reflect.Ptr {
		return nil, InvalidSpecFailure
	}

	s = s.Elem()
	if s.Kind() != reflect.Struct {
		return nil, InvalidSpecFailure
	}

	if len(prefixParam) > 0 {
		prefix = prefixParam[0]
	}

	specType := s.Type()
	configName := specType.Name()
	fmt.Println("-->", configName, "<--")
	fields := make([]Field, 0, s.NumField())
	for i := 0; i < s.NumField(); i++ {
		field := s.Field(i)
		ftype := specType.Field(i)

		confTags := ftype.Tag.Get("conf")
		if !field.CanSet() || confTags == "-" {
			continue
		}

		fieldOpts, err := ParseTag(confTags)
		if err != nil {
			return fields, failure.Wrap(err, "parseTag failed")
		}

		for field.Kind() == reflect.Ptr {
			if field.IsNil() {
				if field.Type().Elem().Kind() != reflect.Struct {
					// nil pointer to a non-struct: leave it alone
					break
				}
				// nil pointer to a struct: create a zero instance
				field.Set(reflect.New(field.Type().Elem()))
			}
			field = field.Elem()
		}

		switch {
		case field.Kind() == reflect.Struct:
			if DecoderFrom(field) == nil &&
				SetterFrom(field) == nil &&
				TextUnmarshaler(field) == nil &&
				BinaryUnmarshaler(field) == nil {

				embeddedPtr := field.Addr().Interface()
				innerFields, err := Fields(embeddedPtr, prefixParam...)
				if err != nil {
					return fields, failure.Wrap(err, "Collect failed for embedded struct")
				}
				fields = append(fields, innerFields...)
			}
		default:
			// capture info about the config variable
			data := Field{
				Name:         ftype.Name,
				Prefix:       prefix,
				ReflectValue: field,
				ReflectTag:   ftype.Tag,
				Tag:          fieldOpts,
			}
			fields = append(fields, data)
		}
	}

	return fields, nil
}

// Decoder has the same semantics as Setter, but takes higher precedence.
// It is provided for historical compatibility.
type Decoder interface {
	Decode(value string) error
}

// Setter is implemented by types can self-deserialize values.
// Any type that implements flag.Value also implements Setter.
type Setter interface {
	Set(value string) error
}

func interfaceFrom(field reflect.Value, fn func(interface{}, *bool)) {
	// it may be impossible for a struct field to fail this check
	if !field.CanInterface() {
		return
	}

	var ok bool
	fn(field.Interface(), &ok)
	if !ok && field.CanAddr() {
		fn(field.Addr().Interface(), &ok)
	}
}

func DecoderFrom(field reflect.Value) (d Decoder) {
	interfaceFrom(field, func(v interface{}, ok *bool) { d, *ok = v.(Decoder) })
	return d
}

func SetterFrom(field reflect.Value) (s Setter) {
	interfaceFrom(field, func(v interface{}, ok *bool) { s, *ok = v.(Setter) })
	return s
}

func TextUnmarshaler(field reflect.Value) (t encoding.TextUnmarshaler) {
	interfaceFrom(field, func(v interface{}, ok *bool) { t, *ok = v.(encoding.TextUnmarshaler) })
	return t
}

func BinaryUnmarshaler(field reflect.Value) (b encoding.BinaryUnmarshaler) {
	interfaceFrom(field, func(v interface{}, ok *bool) { b, *ok = v.(encoding.BinaryUnmarshaler) })
	return b
}
