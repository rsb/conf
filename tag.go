package conf

import (
	"regexp"
	"strings"

	"github.com/rsb/failure"
)

// Tag represents the annotated tag `conf` used to control how we will
// parse that config's property.
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
				if isDefaultValueMapOrList(value) {
					var err error
					value, err = normalizeDefaultValueMapOrList(value)
					if err != nil {
						return tag, failure.Wrap(err, "normalizeDefaultValueMapOrList failed")
					}
				}

				tag.Default = value

			case "env":
				tag.EnvVar = value
			}
		}
	}

	return tag, nil
}

func isDefaultValueMapOrList(value string) bool {
	return strings.Contains(value, "map(") ||
		strings.Contains(value, "list(")
}

func normalizeDefaultValueMapOrList(value string) (string, error) {
	lastChar := value[len(value)-1:]
	if lastChar != ")" {
		return "", failure.Config("tag (default) invalid list or map syntax")
	}

	re := regexp.MustCompile(`\((.*?)\)`)
	matches := re.FindAllString(value, -1)
	for _, elem := range matches {
		elem = strings.Trim(elem, "(")
		elem = strings.Trim(elem, ")")
		value = elem
	}
	value = strings.Replace(value, "|", ":", -1)
	value = strings.Replace(value, ";", ",", -1)
	return value, nil
}
