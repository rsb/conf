package conf_test

import (
	"testing"

	"github.com/rsb/conf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
