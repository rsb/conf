/*
Package conf is a package that specializes in parsing out environment variables
using structs with annotated tags to control how it is done.
*/
package conf

import (
	"os"

	"github.com/k0kubun/pp"

	"github.com/rsb/failure"
)

func ProcessEnv(spec interface{}, prefix ...string) error {
	fields, err := Fields(spec, prefix...)
	pp.Println(fields)
	if err != nil {
		return failure.Wrap(err, "Fields failed")
	}

	for _, field := range fields {
		value, ok := os.LookupEnv(field.EnvVar())
		if !ok && field.IsDefault() {
			value = field.DefaultValue()
		}

		if !ok && !field.IsDefault() {
			if field.IsRequired() {
				return failure.Config("required key (%s,%s) missing value", field.Name, field.EnvVar())
			}
			continue
		}

		if err = ProcessField(value, field.ReflectValue); err != nil {
			return failure.Wrap(err, "ProcessField failed")
		}
	}

	return nil
}

func HandleUnsetEnvVar(field Field) (string, error) {
	if field.IsDefault() {
		return field.DefaultValue(), nil
	}

	if !field.IsRequired() {
		return "", nil
	}

	return "", failure.Config("required key (%s) missing value", field.EnvVar())
}

// EnvVar ensures the variable you are looking for is set. If you don't care
// about that use EnvVarOptional instead
func EnvVar(key string) (string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return value, failure.Config("env var (%s) is not set", key)
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
