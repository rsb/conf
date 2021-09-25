/*
Package conf is a package that specializes in parsing out environment variables
using structs with annotated tags to control how it is done.
*/
package conf

import (
	"os"

	"github.com/rsb/failure"
)

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
