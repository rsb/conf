package conf_test

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rsb/conf"
)

type HonorDecodeInStruct struct {
	Value string
}

func (h *HonorDecodeInStruct) Decode(_ string) error {
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

type Specification struct {
	Embedded
	EmbeddedButIgnored     `conf:"-"`
	Debug                  bool
	Port                   int
	Rate                   float32
	User                   string
	TTL                    uint32
	Timeout                time.Duration
	AdminUsers             []string
	MagicNumbers           []int
	EmptyNumbers           []int
	ByteSlice              []byte
	ColorCodes             map[string]int
	SomePointer            *string
	SomePointerWithDefault *string `conf:"default:foo2baz, desc:foorbar is the word"`
	NoPrefixWithAlt        string  `conf:"env:SERVICE_HOST,no-prefix"`
	DefaultVar             string  `conf:"default:foobar"`
	RequiredVar            string  `conf:"required"`
	NoPrefixDefault        string  `conf:"env:BROKER,default:127.0.0.1"`
	RequiredDefault        string  `conf:"required,default:foo2bar"`
	Ignored                string  `conf:"-"`
	NestedSpecification    struct {
		Property            string
		PropertyWithDefault string `conf:"default:fuzzybydefault"`
	}
	AfterNested  string
	DecodeStruct HonorDecodeInStruct `conf:"env:honor"`
	Datetime     time.Time
	MapField     map[string]string `conf:"default:map(one|two;three|four)"`
	UrlValue     CustomURL
	UrlPointer   *CustomURL
}

type Embedded struct {
	Enabled         bool
	EmbeddedPort    int
	MultiWordVar    string
	EmbeddedIgnored string `conf:"-"`
}

type EmbeddedButIgnored struct {
	FirstEmbeddedButIgnored  string
	SecondEmbeddedButIgnored string
}

type App struct {
	API
	DB
}

type API struct {
	URL      string
	Timeout  time.Duration
	SomeDate time.Time
}

type DB struct {
	Host string
	Name string
	User string
	Port int
	Pass string
}

func TestProcessEnv(t *testing.T) {
	var s Specification
	os.Clearenv()
	setenv(t, "ENV_DEBUG", "true")
	setenv(t, "ENV_PORT", "8080")
	setenv(t, "ENV_RATE", "0.5")
	setenv(t, "ENV_USER", "rsb")
	setenv(t, "ENV_TIMEOUT", "2m")
	setenv(t, "ENV_ADMIN_USERS", "John,Adam,Will")
	setenv(t, "ENV_MAGIC_NUMBERS", "3,5,10,20")
	setenv(t, "ENV_EMPTY_NUMBERS", "")
	setenv(t, "ENV_BYTE_SLICE", "this is a test value")
	setenv(t, "ENV_COLOR_CODES", "red:1,green:2,blue:3")
	setenv(t, "SERVICE_HOST", "127.0.0.1")
	setenv(t, "ENV_TTL", "30")
	setenv(t, "ENV_REQUIRED_VAR", "foo")
	setenv(t, "ENV_IGNORED", "was-not-ignored")
	setenv(t, "ENV_NESTED_SPECIFICATION_PROPERTY", "i_am_nested")
	setenv(t, "ENV_AFTER_NESTED", "after")
	setenv(t, "ENV_HONOR", "honor")
	setenv(t, "ENV_DATETIME", "2016-08-16T18:57:05Z")
	setenv(t, "ENV_URL_VALUE", "https://google.com")
	setenv(t, "ENV_URL_POINTER", "https://google.com")
	err := conf.ProcessEnv(&s, "env")
	require.NoError(t, err)

	assert.Equal(t, "127.0.0.1", s.NoPrefixWithAlt)
	assert.True(t, s.Debug)
	assert.Equal(t, 8080, s.Port)
	assert.Equal(t, float32(0.5), s.Rate)
	assert.Equal(t, uint32(30), s.TTL)
	assert.Equal(t, "rsb", s.User)
	assert.Equal(t, 2*time.Minute, s.Timeout)
	assert.Equal(t, "foo", s.RequiredVar)
	assert.Equal(t, []string{"John", "Adam", "Will"}, s.AdminUsers)
	assert.Equal(t, []int{3, 5, 10, 20}, s.MagicNumbers)
	assert.Empty(t, s.EmptyNumbers)
	assert.Equal(t, []byte("this is a test value"), s.ByteSlice)
	assert.Empty(t, s.Ignored)
	assert.Equal(t, map[string]int{"red": 1, "green": 2, "blue": 3}, s.ColorCodes)
	assert.Equal(t, "i_am_nested", s.NestedSpecification.Property)
	assert.Equal(t, "fuzzybydefault", s.NestedSpecification.PropertyWithDefault)
	assert.Equal(t, "after", s.AfterNested)
	assert.Equal(t, "decoded", s.DecodeStruct.Value)

	expected := time.Date(2016, 8, 16, 18, 57, 05, 0, time.UTC)
	assert.Equal(t, expected, s.Datetime)

	u, err := url.Parse("https://google.com")
	require.NoError(t, err, "url.Parse is not expected to fail")
	assert.Equal(t, u, s.UrlValue.Value)
	assert.Equal(t, u, s.UrlPointer.Value)
}

func TestProcessEnv_CamelCaseSplitUpperToLowerCase(t *testing.T) {
	config := struct {
		FOOBar string
	}{}

	os.Clearenv()
	setenv(t, "FOO_BAR", "foobar")
	err := conf.ProcessEnv(&config)
	require.NoError(t, err)
	assert.Equal(t, "foobar", config.FOOBar)
}

func TestProcessEnv_ParseBoolFailure(t *testing.T) {
	var s Specification
	os.Clearenv()
	setenv(t, "DEBUG", "string")
	setenv(t, "REQUIRED_VAR", "foo")
	err := conf.ProcessEnv(&s)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")

	assert.Contains(t, err.Error(), "ProcessField failed (Debug)")
	assert.Contains(t, err.Error(), "strconv.ParseBool failed")
}

func TestProcessEnv_ParseFloatFailure(t *testing.T) {
	var s Specification
	os.Clearenv()
	setenv(t, "RATE", "string")
	setenv(t, "REQUIRED_VAR", "foo")
	err := conf.ProcessEnv(&s)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")

	assert.Contains(t, err.Error(), "ProcessField failed (Rate)")
	assert.Contains(t, err.Error(), "strconv.ParseFloat failed")
}

func TestProcessEnv_ParseUIntFailure(t *testing.T) {
	var s Specification
	os.Clearenv()
	setenv(t, "TTL", "-30")
	setenv(t, "REQUIRED_VAR", "foo")
	err := conf.ProcessEnv(&s)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")

	assert.Contains(t, err.Error(), "ProcessField failed (TTL)")
	assert.Contains(t, err.Error(), "strconv.ParseUint failed")
}

func TestProcessEnv_RequiredFailure(t *testing.T) {
	var s Specification
	os.Clearenv()
	err := conf.ProcessEnv(&s)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")

	assert.Contains(t, err.Error(), "required key (RequiredVar,REQUIRED_VAR) missing value")
}

func TestProcessEnv_InvalidSpecFailure(t *testing.T) {
	config := struct {
		Foo string `conf:"default:map(ab,d, required"`
	}{}

	os.Clearenv()
	err := conf.ProcessEnv(&config)
	require.Error(t, err, "conf.ProcessEnv is expected to fail")

	assert.Contains(t, err.Error(), "Fields failed")
	assert.Contains(t, err.Error(), "parseTag failed (Foo)")
}

func TestEnvToMap_SuccessSingleStruct(t *testing.T) {
	config := struct {
		ValueA string `conf:"default:foo"`
		ValueB int
	}{}

	os.Clearenv()
	setenv(t, "VALUE_B", "xyz")

	result, err := conf.EnvToMap(&config)
	require.NoError(t, err)

	expected := map[string]string{
		"VALUE_A": "foo",
		"VALUE_B": "xyz",
	}

	assert.Equal(t, expected, result)
}

func TestEnvToMap_SuccessMultipleStructs(t *testing.T) {

	apiURLKey := "API_URL"
	apiURLValue := "some_url_string"

	apiTimeoutKey := "API_TIMEOUT"
	apiTimeoutValue := "5s"

	apiDateKey := "API_SOME_DATE"
	apiDateValue := "some-date"

	dbNameKey := "DB_NAME"
	dbNameValue := "db-name"
	dbUserKey := "DB_USER"
	dbUserValue := "db-user"
	dbHostKey := "DB_HOST"
	dbHostValue := "db-host"
	dbPortKey := "DB_PORT"
	dbPortValue := "5432"
	dbPassKey := "DB_PASS"
	dbPassValue := "password"

	var config App
	os.Clearenv()
	setenv(t, apiURLKey, apiURLValue)
	setenv(t, apiTimeoutKey, apiTimeoutValue)
	setenv(t, apiDateKey, apiDateValue)

	setenv(t, dbNameKey, dbNameValue)
	setenv(t, dbUserKey, dbUserValue)
	setenv(t, dbHostKey, dbHostValue)
	setenv(t, dbPassKey, dbPassValue)
	setenv(t, dbPortKey, dbPortValue)

	result, err := conf.EnvToMap(&config)
	require.NoError(t, err)

	expected := map[string]string{
		apiURLKey:     apiURLValue,
		apiTimeoutKey: apiTimeoutValue,
		apiDateKey:    apiDateValue,
		dbNameKey:     dbNameValue,
		dbUserKey:     dbUserValue,
		dbHostKey:     dbHostValue,
		dbPortKey:     dbPortValue,
		dbPassKey:     dbPassValue,
	}

	assert.Equal(t, expected, result)
}

func TestEnvNames_SuccessMultipleStructs(t *testing.T) {

	apiURLKey := "API_URL"
	apiTimeoutKey := "API_TIMEOUT"
	apiDateKey := "API_SOME_DATE"
	dbNameKey := "DB_NAME"
	dbUserKey := "DB_USER"
	dbHostKey := "DB_HOST"
	dbPortKey := "DB_PORT"
	dbPassKey := "DB_PASS"

	var config App
	os.Clearenv()

	result, err := conf.EnvNames(&config)
	require.NoError(t, err)

	expected := []string{
		apiURLKey,
		apiTimeoutKey,
		apiDateKey,
		dbHostKey,
		dbNameKey,
		dbUserKey,
		dbPortKey,
		dbPassKey,
	}

	assert.Equal(t, expected, result)
}

func setenv(t *testing.T, key, value string) {
	require.NoError(t, os.Setenv(key, value))
}
