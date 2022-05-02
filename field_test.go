package conf_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/rsb/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestField_EnvVariable_NoPrefix(t *testing.T) {
	f := conf.Field{
		EnvVar: "FOO",
		Tag: conf.Tag{
			NoPrefix: true,
		},
	}

	assert.Equal(t, "FOO", f.EnvVariable())
}

func TestField_EnvVariable_WithPrefix(t *testing.T) {
	f := conf.Field{
		Prefix: "MY_PREFIX",
		EnvVar: "FOO",
		Tag:    conf.Tag{},
	}

	assert.Equal(t, "MY_PREFIX_FOO", f.EnvVariable())
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

func TestField_ParamStoreKey(t *testing.T) {
	tests := []struct {
		name    string
		field   conf.Field
		isCheck bool
		value   string
	}{
		{
			name:    "Param store is inside the tag",
			isCheck: true,
			value:   "foo-bar",
			field: conf.Field{
				Tag: conf.Tag{
					PStoreVar: "foo-bar",
				},
			},
		},
		{
			name:    "Param store is not inside the tag",
			isCheck: false,
			value:   "",
			field: conf.Field{
				Tag: conf.Tag{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.field.ParamStoreKey()
			check := tt.field.IsParamStore()
			assert.Equal(t, tt.value, result)
			assert.Equal(t, tt.isCheck, check)
		})
	}
}

func TestField_ViperKey(t *testing.T) {
	tests := []struct {
		name    string
		field   conf.Field
		isCheck bool
		value   string
	}{
		{
			name:    "Viper is inside the tag",
			isCheck: true,
			value:   "foo-bar",
			field: conf.Field{
				Tag: conf.Tag{
					CLIFlag: "foo-bar",
				},
			},
		},
		{
			name:    "Viper is not inside the tag",
			isCheck: false,
			value:   "",
			field: conf.Field{
				Tag: conf.Tag{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.field.CLIFlag()
			check := tt.field.IsCLI()
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

func TestFields_ReflectPtr_Success(t *testing.T) {
	type MyConfig struct {
		FieldA *string `conf:"env:Field"`
	}

	var config MyConfig
	result, err := conf.Fields(&config)
	require.NoError(t, err, "conf.Fields is not expected to fail")
	require.Len(t, result, 1)

	f := result[0]

	assert.True(t, f.ReflectValue.IsNil())
}

func TestFields_ReflectStructPtr_Success(t *testing.T) {
	type Foo struct {
		Item int
	}

	type MyConfig struct {
		FieldA *Foo `conf:"env:Field"`
	}

	var config MyConfig
	result, err := conf.Fields(&config)
	require.NoError(t, err, "conf.Fields is not expected to fail")
	require.Len(t, result, 1)

	f := result[0]
	assert.Equal(t, "int", f.ReflectValue.Kind().String())
}

func TestFields_EmbeddedStruct_Failure(t *testing.T) {
	type Foo struct {
		IsFlag bool `conf:"env:,default:xys"`
	}

	type MyConfig struct {
		Foo
		FieldA string `conf:"env:Field"`
	}

	var config MyConfig
	_, err := conf.Fields(&config)
	require.Error(t, err, "conf.Fields is expected to fail")
	assert.Contains(t, err.Error(), "Fields failed for embedded struct")
}

type Foo struct {
	IsFlag bool `conf:"env:IS_FLAG,default:true"`
}

func (f *Foo) Decode(_ string) error {
	f.IsFlag = true
	return nil
}
func TestFields_EmbeddedStruct_Success(t *testing.T) {
	type MyConfig struct {
		Foo
		FieldA string `conf:"env:Field"`
	}

	var config MyConfig
	result, err := conf.Fields(&config)
	require.NoError(t, err, "conf.Fields is not expected to fail")
	require.Len(t, result, 2)

	a := result[0]
	assert.Equal(t, "Foo", a.Name)

	b := result[1]
	assert.Equal(t, "FieldA", b.Name)

}

/*
env:aaa; default:a,b,c,d; xyx;
*/
func TestTextUnmarshaler(t *testing.T) {

	config := struct {
		TimeValue time.Time
	}{}

	timeValue := "2016-08-16T18:57:05Z"

	field := reflect.ValueOf(&config).Elem().Field(0)

	tu := conf.TextUnmarshaler(field)
	require.NotNil(t, tu)

	err := tu.UnmarshalText([]byte(timeValue))
	require.NoError(t, err, "tu.UnmarshalText is not expected to fail")

	var expected time.Time
	err = expected.UnmarshalText([]byte(timeValue))
	assert.Equal(t, expected, config.TimeValue)
}
