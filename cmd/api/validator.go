package main

type validator struct {
	errors map[string]string
}

func (v *validator) valid() bool {
	return len(v.errors) == 0
}

func (v *validator) addError(key, message string) {
	if _, ok := v.errors[key]; !ok {
		v.errors[key] = message
	}
}

func (v *validator) check(ok bool, key, message string) {
	if !ok {
		v.addError(key, message)
	}
}
