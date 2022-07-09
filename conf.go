/*
Package conf is a package that specializes in parsing out environment variables
using structs with annotated tags to control how it is done.
*/
package conf

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/rsb/failure"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	AWSLambdaFunctionNameVar = "AWS_LAMBDA_FUNCTION_NAME"
	AWSProfile               = "AWS_PROFILE"
	AWSRegion                = "AWS_REGION"
	AppName                  = "APP_NAME"
	GlobalParamStoreKey      = "global"
)

var excludedVars = []string{
	AppName,
	AWSProfile,
	AWSRegion,
	AWSLambdaFunctionNameVar,
}

type Config struct {
	Data        interface{}
	SkipDefault bool
	Prefix      string
}

func NewConfig(d interface{}, prefixOpt ...string) *Config {
	prefix := ""
	if len(prefixOpt) > 0 && prefixOpt[0] != "" {
		prefix = prefixOpt[0]
	}
	return &Config{Data: d, SkipDefault: true, Prefix: prefix}
}

func (c *Config) GetPrefix() string {
	return c.Prefix
}

func (c *Config) SetPrefix(prefix string) {
	c.Prefix = prefix
}

func (c *Config) IsPrefixEnabled() bool {
	return c.Prefix != ""
}

func (c *Config) loadPrefix() []string {
	if !c.IsPrefixEnabled() {
		return []string{}
	}

	return []string{c.GetPrefix()}
}

func (c *Config) MarkDefaultsAsExcluded() {
	c.SkipDefault = true
}

func (c *Config) MarkDefaultsAsIncluded() {
	c.SkipDefault = false
}

func (c *Config) SetExcludeDefaults(value bool) {
	c.SkipDefault = value
}

func (c *Config) IsDefaultsExcluded() bool {
	return c.SkipDefault
}

func (c *Config) ProcessCLI(cmd *cobra.Command, v *viper.Viper) error {
	if err := ProcessCLI(cmd, v, c.Data, c.loadPrefix()...); err != nil {
		return failure.Wrap(err, "ProcessCLI failed")
	}

	return nil
}

func (c *Config) ProcessEnv() error {
	if err := ProcessEnv(c.Data, c.loadPrefix()...); err != nil {
		return failure.Wrap(err, "ProcessEnv failed")
	}

	return nil
}

func (c *Config) CollectParamsFromEnv(appTitle string) (map[string]string, error) {
	result, err := CollectParamsFromEnv(appTitle, c.Data, c.SkipDefault, c.loadPrefix()...)
	if err != nil {
		return nil, failure.Wrap(err, "CollectParamsFromEnv failed")
	}

	return result, nil
}

func (c *Config) ParamNames(appTitle string) ([]string, error) {
	name, err := ParamNames(appTitle, c.Data, c.IsDefaultsExcluded(), c.loadPrefix()...)
	if err != nil {
		return nil, failure.Wrap(err, "EnvNames failed")
	}

	return name, nil
}

func (c *Config) EnvNames() ([]string, error) {
	name, err := EnvNames(c.Data, c.loadPrefix()...)
	if err != nil {
		return nil, failure.Wrap(err, "EnvNames failed")
	}

	return name, nil
}

func (c *Config) EnvToMap() (map[string]string, error) {
	result, err := EnvToMap(c.Data, c.loadPrefix()...)
	if err != nil {
		return nil, failure.Wrap(err, "EnvToMap failed")
	}

	return result, nil
}

func (c *Config) EnvReport() (map[string]string, error) {
	result, err := EnvReport(c.Data, c.loadPrefix()...)
	if err != nil {
		return nil, failure.Wrap(err, "Report failed")
	}

	return result, nil
}

func BindCLI(cmd *cobra.Command, v *viper.Viper, spec interface{}, prefix ...string) error {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return failure.Wrap(err, "Fields failed")
	}

	for _, field := range fields {
		if !field.IsCLI() {
			continue
		}

		flag := field.CLIFlag()
		short := field.CLIShortFlag()
		usage := field.CLIUsage()
		defaultValue := field.DefaultValue()

		flagSet := cmd.Flags()
		if field.IsPersistentFlag() {
			flagSet = cmd.PersistentFlags()
		}

		switch field.ReflectValue.Type().Kind() {
		case reflect.Bool:
			if defaultValue == "" {
				defaultValue = "false"
			}
			dv, err := strconv.ParseBool(defaultValue)
			if err != nil {
				return failure.ToSystem(err, "strconv.ParseBool failed")
			}
			if short != "" {
				flagSet.BoolP(flag, short, dv, usage)
			} else {
				flagSet.Bool(flag, dv, usage)
			}
		default:
			if short != "" {
				flagSet.StringP(flag, short, defaultValue, usage)
			} else {
				flagSet.String(flag, defaultValue, usage)
			}
		}

		lookupFlag := flagSet.Lookup(flag)
		flagID := field.BindName()

		if err = v.BindPFlag(flagID, lookupFlag); err != nil {
			return failure.ToSystem(err, "v.BindPFlag failed for (%s)", flag)
		}
	}

	return nil
}

func ProcessCLI(cmd *cobra.Command, v *viper.Viper, spec interface{}, prefix ...string) error {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return failure.Wrap(err, "Fields failed")
	}

	var failed *failure.Multi
	for _, field := range fields {
		var value string
		env := field.EnvVariable()
		flag := field.CLIFlag()
		flagID := field.BindName()

		f := cmd.Flags().Lookup(flag)
		// CLI flag has the highest priority
		if flag != "" && f != nil && f.Value.String() != "" && f.Changed {
			value = f.Value.String()

		} else if env != "" {
			var ok bool
			if env != "-" {
				// Env is the 2nd highest priority
				value, ok = os.LookupEnv(env)

				if !ok {
					value, _ = fromViper(v, flagID)
				}
			} else {
				// Env is ignored, but we still need to check inside a config file
				value, _ = fromViper(v, flagID)
			}
		}

		// This will not happen if you use BindCLI because the default value is
		// always set. It is here just in case you are doing things manually
		if value == "" {
			if field.IsDefault() {
				value = field.DefaultValue()
			} else {
				if field.IsRequired() {
					failed = failure.Append(failed, failure.Config("required key (field:%s,env:%s,cli:%s) missing value", field.Name, env, flag))
					continue
				}
			}
		}

		if err = ProcessField(value, field.ReflectValue); err != nil {
			err = failure.Wrap(err, "ProcessField failed (%s)", field.Name)
			failed = failure.Append(failed, err)
			continue
		}
	}

	return failed.ErrorOrNil()
}

func fromViper(v *viper.Viper, flagID string) (string, bool) {
	var value string
	var found bool

	if v.InConfig(flagID) {
		found = true
		data := v.Get(flagID)
		switch d := data.(type) {
		case map[string]interface{}:
			for k, v := range d {
				value += fmt.Sprintf("%s:%s,", k, v)
			}
			value = strings.TrimRight(value, ",")
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			value = fmt.Sprintf("%d", d)
		case float32, float64:
			value = fmt.Sprintf("%f", d)
		case string:
			value = fmt.Sprintf("%s", d)
		case bool:
			value = fmt.Sprintf("%t", d)
		default:
			value = fmt.Sprintf("%v", d)
		}
	}

	return value, found
}

func ProcessEnv(spec interface{}, prefix ...string) error {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return failure.Wrap(err, "Fields failed")
	}

	for _, field := range fields {
		env := field.EnvVariable()
		if env == "" {
			return failure.System("env: is required but empty for (%s)", field.Name)
		}

		value, ok := os.LookupEnv(env)
		if !ok && field.IsDefault() {
			value = field.DefaultValue()
		}

		if !ok && !field.IsDefault() {
			if field.IsRequired() {
				return failure.Config("required key (%s,%s) missing value", field.Name, env)
			}
			continue
		}

		if err = ProcessField(value, field.ReflectValue); err != nil {
			return failure.Wrap(err, "ProcessField failed (%s)", field.Name)
		}
	}

	return nil
}

func PStoreKey(field Field, appTitle, env string) string {
	var key string
	pkey := field.ParamStoreKey()
	switch {
	case pkey != "":
		key = pkey
	case field.IsGlobalParamStore():
		key = fmt.Sprintf("/%s/%s", GlobalParamStoreKey, env)
	default:
		key = fmt.Sprintf("/%s/%s", appTitle, env)
	}

	return key
}

func CollectParamsFromEnv(appTitle string, spec interface{}, skipDefaults bool, prefix ...string) (map[string]string, error) {
	if appTitle == "" {
		return nil, failure.System("appTitle is empty")
	}

	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

	result := map[string]string{}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		key := PStoreKey(field, appTitle, env)

		if env == "-" || key == "-" {
			continue
		}

		if env == "" {
			return result, failure.System("env: is required but empty for (%s)", field.Name)
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}

		value, ok := os.LookupEnv(env)
		if !ok {
			if field.IsDefault() {
				if skipDefaults {
					continue
				}
				value = field.DefaultValue()
			} else if field.IsRequired() {
				return result, failure.Config("required key (%s,%s) missing value", field.Name, env)
			}
		}

		result[key] = value
	}

	return result, nil
}

func ParamEnvField(appTitle, env string, field Field) (string, string, error) {
	key := PStoreKey(field, appTitle, env)
	value, ok := os.LookupEnv(env)
	if !ok && field.IsDefault() {
		value = field.DefaultValue()
	}

	if !ok && !field.IsDefault() {
		if field.IsRequired() {
			return key, value, failure.Config("required key (%s,%s) missing value", field.Name, env)
		}
	}

	return key, value, nil
}

func ParamNames(appTitle string, spec interface{}, skipDefaults bool, prefix ...string) ([]string, error) {
	if appTitle == "" {
		return nil, failure.System("appTitle is empty")
	}

	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

	var result []string

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		key := PStoreKey(field, appTitle, env)

		if env == "-" || key == "-" {
			continue
		}

		if env == "" {
			return result, failure.System("env: is required but empty for (%s)", field.Name)
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}

		if skipDefaults && field.IsDefault() {
			continue
		}

		result = append(result, key)
	}

	return result, nil
}

func EnvReport(spec interface{}, prefix ...string) (map[string]string, error) {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

	result := map[string]string{}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		if env == "-" {
			continue
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}

		if env == "" {
			return result, failure.System("env: is required but empty for (%s)", field.Name)
		}

		value, ok := os.LookupEnv(env)
		if !ok && field.IsDefault() {
			value = field.DefaultValue()
		}

		result[env] = value
	}

	return result, nil
}

func EnvToMap(spec interface{}, prefix ...string) (map[string]string, error) {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

	result := map[string]string{}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		if env == "-" {
			continue
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}

		if env == "" {
			return result, failure.System("env: is required but empty for (%s)", field.Name)
		}

		value, ok := os.LookupEnv(env)
		if !ok && field.IsDefault() {
			value = field.DefaultValue()
		}

		if !ok && !field.IsDefault() {
			if field.IsRequired() {
				return result, failure.Config("required key (%s,%s) missing value", field.Name, env)
			}
		}

		result[env] = value
	}

	return result, nil
}

func EnvNamesNoDefaults(spec interface{}, prefix ...string) ([]string, error) {
	var names []string

	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		if env == "-" || field.IsDefault() {
			continue
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}
		names = append(names, env)
	}

	return names, nil
}

func EnvNames(spec interface{}, prefix ...string) ([]string, error) {
	var names []string

	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

OUTER:
	for _, field := range fields {
		env := field.EnvVariable()
		if env == "-" {
			continue
		}

		for _, ev := range excludedVars {
			if env == ev {
				continue OUTER
			}
		}
		names = append(names, env)
	}

	return names, nil
}

// EnvVar ensures the variable you are looking for is set. If you don't care
// about that use EnvVarOptional instead
func EnvVar(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return value, failure.NotFound("env var (%s) is not set", key)
	}

	return value, nil
}

// EnvVarStrict ensures the variable is set and not empty
func EnvVarStrict(key string) (string, error) {
	value, err := EnvVar(key)
	if err != nil {
		return value, failure.Wrap(err, "EnvVar failed")
	}

	if value == "" {
		return value, failure.Config("env var (%s) is empty", key)
	}

	return value, nil
}

// EnvVarOptional is a wrapper around os.Getenv with the intent that by using
// this method you are declaring in code that you don't care about empty
// env vars. This is better than just using os.Getenv because that intent
// is not conveyed. So this simple wrapper has the purpose of reveal intent
// and not wrapping for the sake of wrapping
func EnvVarOptional(key string) string {
	return os.Getenv(key)
}
