package validator

import (
	"regexp"
	"slices"
)

var (
	EmailRX = regexp.MustCompile(
		"^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$",
	)
)

type Validator struct {
	Errors map[string]string
}

// New return a new Validator instance
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// Vaild checks if all the validations pass and there are no errors
func (v *Validator) Vaild() bool {
	return len(v.Errors) == 0
}

// AddError adds the given error message on Errors map at the given key provided
// the key doesn't exist already on Errors
func (v *Validator) AddError(key, message string) {
	if _, exists := v.Errors[key]; !exists {
		v.Errors[key] = message
	}
}

// Check cheks if the condition fails and adds the errror message and key if it
// does
func (v *Validator) Check(condition bool, key, message string) {
	if !condition {
		v.Errors[key] = message
	}
}

// ValueInList checks if a specific string is in a list of strings
func ValueInList(value string, list ...string) bool {
	return slices.Contains(list, value)
}

// Unique checks if a given string list doesn't have any duplicates
func Unique(values []string) bool {
	uniqueValues := make(map[string]bool)
	for _, value := range values {
		if _, exists := uniqueValues[value]; exists {
			return false
		}
		uniqueValues[value] = true
	}
	return true
}

func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}
