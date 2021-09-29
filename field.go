package conf

import (
	"encoding"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/rsb/failure"
)

const (
	classLower int = iota
	classUpper
	classNumber
	classOther
)

var (
	InvalidSpecFailure = failure.Config("specification must be a struct pointer")
)

// Field holds information about the current configuration variable
type Field struct {
	Name         string
	EnvKey       []string
	envVar       string
	ReflectValue reflect.Value
	ReflectTag   reflect.StructTag
	Tag          Tag
}

func (f Field) PropertyName() string {
	return f.Name
}

func (f Field) EnvVar() string {
	return f.envVar
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
	var prefix []string
	var fields []Field
	s := reflect.ValueOf(spec)

	if s.Kind() != reflect.Ptr {
		return fields, InvalidSpecFailure
	}

	s = s.Elem()
	if s.Kind() != reflect.Struct {
		return fields, InvalidSpecFailure
	}

	if len(prefixParam) > 0 {
		prefix = prefixParam
	}

	specType := s.Type()

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		ftype := specType.Field(i)

		confTags := ftype.Tag.Get("conf")
		if !f.CanSet() || confTags == "-" {
			continue
		}

		fieldName := ftype.Name

		fieldOpts, err := ParseTag(confTags)
		if err != nil {
			return fields, failure.Wrap(err, "parseTag failed (%s)", fieldName)
		}

		// Generate the field key. This could be ignored.
		fieldKey := append(prefix, CamelSplit(fieldName)...)
		for f.Kind() == reflect.Ptr {
			if f.IsNil() {
				if f.Type().Elem().Kind() != reflect.Struct {
					// nil pointer to a non-struct: leave it alone
					break
				}
				// nil pointer to a struct: create a zero instance
				f.Set(reflect.New(f.Type().Elem()))
			}
			f = f.Elem()
		}

		switch {
		case f.Kind() == reflect.Struct:
			if DecoderFrom(f) == nil && SetterFrom(f) == nil && TextUnmarshaler(f) == nil && BinaryUnmarshaler(f) == nil {
				innerPrefix := fieldKey
				embeddedPtr := f.Addr().Interface()
				innerFields, err := Fields(embeddedPtr, innerPrefix...)
				if err != nil {
					return fields, failure.Wrap(err, "Collect failed for embedded struct")
				}
				fields = append(fields, innerFields...)
				continue
			}
			data := NewField(fieldName, fieldKey, f, ftype.Tag, fieldOpts)
			fields = append(fields, data)

		default:
			data := NewField(fieldName, fieldKey, f, ftype.Tag, fieldOpts)
			fields = append(fields, data)
		}

	}

	return fields, nil
}

func NewField(name string, fieldKey []string, v reflect.Value, t reflect.StructTag, opts Tag) Field {
	envKey := fieldKey
	if opts.EnvVar != "" {
		if opts.NoPrefix {
			envKey = strings.Split(opts.EnvVar, "_")
		} else {
			envPrefix := envKey[0]
			tmp := strings.Split(opts.EnvVar, "_")
			envKey = append([]string{envPrefix}, tmp...)
		}
	}

	return Field{
		Name:         name,
		EnvKey:       envKey,
		envVar:       strings.ToUpper(strings.Join(envKey, "_")),
		ReflectValue: v,
		ReflectTag:   t,
		Tag:          opts,
	}
}
func ProcessField(value string, field reflect.Value) error {
	typ := field.Type()

	if decoder := DecoderFrom(field); decoder != nil {
		if err := decoder.Decode(value); err != nil {
			return failure.ToSystem(err, "decoder.Decode failed (%s)", value)
		}
		return nil
	}

	// look for Set method if Decode is not defined
	if setter := SetterFrom(field); setter != nil {
		if err := setter.Set(value); err != nil {
			return failure.ToSystem(err, "setter.Set failed (%s)", value)
		}
		return nil
	}

	if t := TextUnmarshaler(field); t != nil {
		if err := t.UnmarshalText([]byte(value)); err != nil {
			return failure.ToSystem(err, "t.UnmarshalText failed (%s)", value)
		}
		return nil
	}

	if b := BinaryUnmarshaler(field); b != nil {
		if err := b.UnmarshalBinary([]byte(value)); err != nil {
			return failure.ToSystem(err, "b.UnmarshalBinary failed (%s)", value)
		}
		return nil
	}

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		if field.IsNil() {
			field.Set(reflect.New(typ))
		}
		field = field.Elem()
	}

	switch typ.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var val int64
		var err error
		if field.Kind() == reflect.Int64 && typ.PkgPath() == "time" && typ.Name() == "Duration" {

			var d time.Duration
			d, err = time.ParseDuration(value)
			if err != nil {
				return failure.ToSystem(err, "time.Duration failed, failed to parse int")
			}
			val = int64(d)
		} else {
			val, err = strconv.ParseInt(value, 0, typ.Bits())
			if err != nil {
				return failure.ToSystem(err, "strconv.ParseInt failed")
			}
		}
		field.SetInt(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(value, 0, typ.Bits())
		if err != nil {
			return failure.ToSystem(err, "strconv.ParseUint failed")
		}
		field.SetUint(val)

	case reflect.Bool:
		val, err := strconv.ParseBool(value)
		if err != nil {
			return failure.ToSystem(err, "strconv.ParseBool failed")
		}
		field.SetBool(val)

	case reflect.Float64, reflect.Float32:
		val, err := strconv.ParseFloat(value, typ.Bits())
		if err != nil {
			return failure.ToSystem(err, "strconv.ParseFloat failed")
		}
		field.SetFloat(val)
	case reflect.Slice:
		sl := reflect.MakeSlice(typ, 0, 0)
		if typ.Elem().Kind() == reflect.Uint8 {
			sl = reflect.ValueOf([]byte(value))
		} else if len(strings.TrimSpace(value)) != 0 {
			vals := strings.Split(value, ",")
			sl = reflect.MakeSlice(typ, len(vals), len(vals))
			for i, val := range vals {
				err := ProcessField(val, sl.Index(i))
				if err != nil {
					return failure.Wrap(err, "processField failed at (%d)", i)
				}
			}
		}
		field.Set(sl)
	case reflect.Map:
		mp := reflect.MakeMap(typ)
		if len(strings.TrimSpace(value)) != 0 {
			pairs := strings.Split(value, ",")
			for _, pair := range pairs {
				kvpair := strings.Split(pair, ":")
				if len(kvpair) != 2 {
					return failure.System("invalid map item: (pair: %q)", pair)
				}

				k := reflect.New(typ.Key()).Elem()
				err := ProcessField(kvpair[0], k)
				if err != nil {
					return failure.Wrap(err, "processField failed for key (pair: %q) ", pair)
				}
				v := reflect.New(typ.Elem()).Elem()
				err = ProcessField(kvpair[1], v)
				if err != nil {
					return failure.Wrap(err, "processField failed for value (pair: %q)", pair)
				}
				mp.SetMapIndex(k, v)
			}
		}
		field.Set(mp)
	}

	return nil
}

// CamelSplit takes a string based on a camel case and splits it.
func CamelSplit(src string) []string {
	if src == "" {
		return []string{}
	}

	if len(src) < 2 {
		return []string{src}
	}

	runes := []rune(src)

	lastClass := charClass(runes[0])
	lastIdx := 0
	var out []string

	for i, r := range runes {
		class := charClass(r)

		// If class has transitioned.
		if class != lastClass {
			// If going from uppercase to lowercase, we want to retain the last
			// uppercase letter for names like FOOBar, which should be split to
			// FOO BAR
			switch {
			case lastClass == classUpper && class != classNumber:
				if i-lastIdx > 1 {
					out = append(out, string(runes[lastIdx:i-1]))
					lastIdx = i - 1
				}
			default:
				out = append(out, string(runes[lastIdx:i]))
				lastIdx = i
			}
		}

		if i == len(runes)-1 {
			out = append(out, string(runes[lastIdx:]))
		}
		lastClass = class
	}

	return out
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

func charClass(r rune) int {
	var result int

	switch {
	case unicode.IsLower(r):
		result = classLower
	case unicode.IsUpper(r):
		result = classUpper
	case unicode.IsDigit(r):
		result = classNumber
	default:
		result = classOther
	}

	return result
}
