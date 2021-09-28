package conf_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/rsb/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
