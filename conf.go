/*
Package conf is a package that specializes in parsing out environment variables
using structs with annotated tags to control how it is done.
*/
package conf

import (
	"os"

	"github.com/rsb/failure"
)

func ProcessEnv(spec interface{}, prefix ...string) error {
	fields, err := Fields(spec, prefix...)
	//pp.Println(fields)
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
			return failure.Wrap(err, "ProcessField failed (%s)", field.Name)
		}
	}

	return nil
}

func EnvToMap(spec interface{}, prefix ...string) (map[string]string, error) {
	fields, err := Fields(spec, prefix...)
	if err != nil {
		return nil, failure.Wrap(err, "Fields failed")
	}

	result := map[string]string{}
	for _, field := range fields {
		envVar := field.EnvVar()
		value, ok := os.LookupEnv(envVar)
		if !ok && field.IsDefault() {
			value = field.DefaultValue()
		}

		if !ok && !field.IsDefault() {
			if field.IsRequired() {
				return result, failure.Config("required key (%s,%s) missing value", field.Name, field.EnvVar())
			}
			continue
		}

		result[envVar] = value
	}

	return result, nil
}
