package conf_test

import (
	"testing"

	"github.com/rsb/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTag_Success(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected conf.Tag
	}{
		{
			name:     "empty",
			tag:      "",
			expected: conf.Tag{},
		},
		{
			name: "minimum info required",
			tag:  "env:FOO_BAR",
			expected: conf.Tag{
				EnvVar:    "FOO_BAR",
				Default:   "",
				IsDefault: false,
				NoPrint:   false,
				NoPrefix:  false,
				Required:  false,
				Mask:      false,
			},
		},
		{
			name: "env and required only",
			tag:  "env:FOO_BAR,required",
			expected: conf.Tag{
				EnvVar:    "FOO_BAR",
				Default:   "",
				IsDefault: false,
				NoPrint:   false,
				NoPrefix:  false,
				Required:  true,
				Mask:      false,
			},
		},
		{
			name: "env and mask only",
			tag:  "env:FOO_BAR,mask",
			expected: conf.Tag{
				EnvVar:    "FOO_BAR",
				Default:   "",
				IsDefault: false,
				NoPrint:   false,
				NoPrefix:  false,
				Required:  false,
				Mask:      true,
			},
		},
		{
			name: "env and no-print only",
			tag:  "env:FOO_BAR,no-print",
			expected: conf.Tag{
				EnvVar:    "FOO_BAR",
				Default:   "",
				IsDefault: false,
				NoPrint:   true,
				NoPrefix:  false,
				Required:  false,
				Mask:      false,
			},
		},
		{
			name: "env and no-prefix only",
			tag:  "env:FOO_BAR,no-prefix",
			expected: conf.Tag{
				EnvVar:    "FOO_BAR",
				Default:   "",
				IsDefault: false,
				NoPrint:   false,
				NoPrefix:  true,
				Required:  false,
				Mask:      false,
			},
		},
		{
			name: "env and default only",
			tag:  "env:FOO_BAR,default:XYZ",
			expected: conf.Tag{
				EnvVar:    "FOO_BAR",
				Default:   "XYZ",
				IsDefault: true,
				NoPrint:   false,
				NoPrefix:  false,
				Required:  false,
				Mask:      false,
			},
		},
		{
			name: "all settings",
			tag:  "env:FOO_BAR,default:XYZ,no-print,mask,no-prefix,required",
			expected: conf.Tag{
				EnvVar:    "FOO_BAR",
				Default:   "XYZ",
				IsDefault: true,
				NoPrint:   true,
				NoPrefix:  true,
				Required:  true,
				Mask:      true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := conf.ParseTag(tt.tag)
			require.NoError(t, err, "conf.ParseTag is not expected to fail")
			assert.Equal(t, tt.expected, result)
		})
	}
}
func TestParseTag_Failures(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		msg  string
	}{
		{
			name: "default without a value",
			tag:  "env:FOO_BAR,default:,required",
			msg:  `tag ("default") missing a value`,
		},
		{
			name: "env without a value",
			tag:  "env:,default:SomeValue,required",
			msg:  `tag ("env") missing a value`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := conf.ParseTag(tt.tag)
			require.Error(t, err, "conf.ParseTag is expected to fail")
			assert.Contains(t, err.Error(), tt.msg)
		})
	}
}
