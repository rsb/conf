package conf

import (
	"strings"

	"github.com/rsb/failure"
)

// Tag represents the annotated tag `conf` used to control how we will
// parse that struct's configuration property.
type Tag struct {
	EnvVar    string
	Default   string
	IsDefault bool
	NoPrint   bool
	NoPrefix  bool
	Required  bool
	Mask      bool
}

func ParseTag(t string) (Tag, error) {
	var tag Tag

	if t == "" {
		return tag, nil
	}

	parts := strings.Split(t, ",")
	for _, part := range parts {
		vals := strings.SplitN(part, ":", 2)
		property := vals[0]
		switch len(vals) {
		case 1:
			switch property {
			case "no-print":
				tag.NoPrint = true
			case "no-prefix":
				tag.NoPrefix = true
			case "required":
				tag.Required = true
			case "mask":
				tag.Mask = true
			}
		case 2:
			value := strings.TrimSpace(vals[1])
			if value == "" {
				return tag, failure.Config("tag (%q) missing a value", property)
			}
			switch property {
			case "default":
				tag.IsDefault = true
				tag.Default = value
			case "env":
				tag.EnvVar = value
			}
		}
	}

	return tag, nil
}
