package validator

import (
	"regexp"
	"slices"
)

var EmailRX = regexp.MustCompile(
	"^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$",
)

type Validator struct {
	Errors map[string]string
}

func NewValidator() *Validator {
	return &Validator{
		Errors: make(map[string]string),
	}
}

func (v *Validator) IsValid() bool {
	return len(v.Errors) == 0
}

func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

func (v *Validator) CheckAdd(condition bool, key, message string) {
	if !condition {
		v.AddError(key, message)
	}
}

func ValueInList(value string, list ...string) bool {
	return slices.Contains(list, value)
}

func ListUnique(list ...string) bool {
	seen := make(map[string]struct{}, len(list))
	for _, value := range list {
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}
