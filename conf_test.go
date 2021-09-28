package conf_test

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/k0kubun/pp"
	"github.com/rsb/conf"
	"github.com/stretchr/testify/require"
)

type HonorDecodeInStruct struct {
	Value string
}

func (h *HonorDecodeInStruct) Decode(env string) error {
	h.Value = "decoded"
	return nil
}

type CustomURL struct {
	Value *url.URL
}

func (cu *CustomURL) UnmarshalBinary(data []byte) error {
	u, err := url.Parse(string(data))
	cu.Value = u
	return err
}

type Spec struct {
	Embedded
	EmbeddedButIgnored `conf:"-""`
	Nested             struct {
		NestValue string `conf:"required"`
	}
	Debug        bool                `conf:"env:DEBUG,default:false"`
	Rate         float32             `conf:"env:RATE,required"`
	Port         int                 `conf:"env:PORT,default:5432"`
	SomeDate     time.Time           `conf:"env:SOME_DATE"`
	MapField     map[string]string   `conf:"env:SOME_MAP,default:one:two,three:four"`
	DecodeStruct HonorDecodeInStruct `conf:"env:HONOR"`
	URLValue     CustomURL           `conf:"env:CUSTOM_URL"`
	URLPtr       *CustomURL          `conf:"env:CUSTOM_URL_PTR"`
}

type EmbeddedButIgnored struct {
	FirstEmbeddedButIgnored  string
	SecondEmbeddedButIgnored string
}

type Embedded struct {
	Enabled      bool `conf:"env:ENABLED,default:true"`
	EmbeddedPort int  `conf:"env:PORT,default:123"`
}

func TestProcessEnv_Success(t *testing.T) {
	var s Spec
	os.Setenv("PREFIX_NESTED_NEST_VALUE", "some-nested-value")
	os.Setenv("PREFIX_DEBUG", "true")
	os.Setenv("PREFIX_RATE", "1.23")
	os.Setenv("PREFIX_SOME_DATE", "2016-08-16T18:57:05Z")
	os.Setenv("PREFIX_SOME_MAP", "five:six,seven:eight")
	os.Setenv("PREFIX_HONOR", "honor")
	os.Setenv("PREFIX_CUSTOM_URL", "https://google.com")
	os.Setenv("PREFIX_CUSTOM_URL_PTR", "https://google.com")
	os.Setenv("PREFIX_FOO", "2016-08-16T18:57:05Z")

	err := conf.ProcessEnv(&s, "PREFIX")
	require.NoError(t, err, "conf.ProcessEnv is not expected to fail")
	pp.Println(s)
}
