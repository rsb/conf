package conf_test

import (
	"reflect"
	"testing"

	"github.com/k0kubun/pp"
	"github.com/rsb/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestField_EnvKey(t *testing.T) {
	tests := []struct {
		name     string
		field    conf.Field
		expected string
	}{
		{
			name: "no prefix",
			field: conf.Field{
				EnvName: "FOO_BAR",
			},
			expected: "FOO_BAR",
		},
		{
			name: "with prefix",
			field: conf.Field{
				Prefix:  "APP_NAME",
				EnvName: "FOO_BAR",
			},
			expected: "APP_NAME_FOO_BAR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.field.EnvKey()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestField_DefaultValue(t *testing.T) {
	tests := []struct {
		name    string
		field   conf.Field
		isCheck bool
		value   string
	}{
		{
			name:    "with default value",
			isCheck: true,
			value:   "some default value",
			field: conf.Field{
				Tag: conf.Tag{
					Default:   "some default value",
					IsDefault: true,
				},
			},
		},
		{
			name:    "without default value",
			isCheck: false,
			value:   "",
			field: conf.Field{
				Tag: conf.Tag{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.field.DefaultValue()
			check := tt.field.IsDefault()
			assert.Equal(t, tt.value, result)
			assert.Equal(t, tt.isCheck, check)
		})
	}
}

func TestField_IsRequired(t *testing.T) {
	field := conf.Field{Tag: conf.Tag{Required: true}}
	assert.True(t, field.IsRequired())

	field = conf.Field{Tag: conf.Tag{}}
	assert.False(t, field.IsRequired())
}

func TestFields_InvalidSpecFailure(t *testing.T) {
	type InvalidConfig struct {
		Value1 string `conf:"env:FOO"`
	}
	var config1 InvalidConfig

	_, err := conf.Fields(config1, "MY_APP")
	require.Error(t, err, "conf.Fields is expected to fail")
	assert.Equal(t, conf.InvalidSpecFailure, err)

	type InvalidConfigString string

	var config2 InvalidConfigString
	_, err = conf.Fields(&config2, "MY_APP")
	require.Error(t, err, "conf.Fields is expected to fail")
	assert.Equal(t, conf.InvalidSpecFailure, err)
}

func TestFields_AllIgnores(t *testing.T) {
	type ConfigAllIgnore struct {
		Value1 string `conf:"-"`
		Value2 int    `conf:"-"`
		Value3 bool   `conf:"-"`
	}

	var config ConfigAllIgnore

	result, err := conf.Fields(&config, "MY_APP")
	require.NoError(t, err, "config.Fields is not expected to fail")
	assert.Empty(t, result)
}

func TestFields_PtrToNonStructDoNothing(t *testing.T) {
	type ConfigDoNothing struct {
		Foo *bool `conf:"env:BAR"`
	}

	var config ConfigDoNothing

	result, err := conf.Fields(&config, "MY_APP")
	require.NoError(t, err, "config.Fields is not expected to fail")
	assert.Equal(t, 1, len(result))

	tag := conf.Tag{EnvVar: "BAR"}
	field := result[0]
	testField(t, result[0], "MY_APP", tag)

	rv := reflect.ValueOf(&config)
	testFieldReflect(t, rv, field, 0)
}

func TestFields_PtrToStruct(t *testing.T) {
	type Config struct {
		Web struct {
			Foo string `conf:"env:FOO"`
		}
	}

	var config Config

	result, err := conf.Fields(&config, "MY_APP")
	require.NoError(t, err, "config.Fields is not expected to fail")
	assert.Equal(t, 1, len(result))

	tag := conf.Tag{EnvVar: "FOO"}
	field := result[0]
	testField(t, result[0], "MY_APP", tag)

	rv := reflect.ValueOf(&config.Web)
	testFieldReflect(t, rv, field, 0)

	pp.Println(result)
}

func testField(t *testing.T, f conf.Field, prefix string, tag conf.Tag) {
	t.Helper()

	assert.Equal(t, prefix, f.Prefix)
	assert.Equal(t, tag, f.Tag)

}
func testFieldReflect(t *testing.T, rv reflect.Value, field conf.Field, fieldNbr int) {
	t.Helper()
	require.Equal(t, rv.Kind(), reflect.Ptr)

	rv = rv.Elem()
	require.Equal(t, rv.Kind(), reflect.Struct)

	rvType := rv.Type()
	f := rv.Field(fieldNbr)
	ftype := rvType.Field(fieldNbr)

	assert.Equal(t, ftype.Name, field.Name)
	assert.Equal(t, ftype.Tag, field.ReflectTag)
	assert.Equal(t, f, field.ReflectValue)
}
