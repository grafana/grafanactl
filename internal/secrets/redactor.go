package secrets

import (
	"reflect"
	"strings"
)

const (
	dataPolicyTag = "datapolicy"
	redacted      = "**REDACTED**"
)

func Redact[V any](value *V) error {
	return redactSecrets(reflect.ValueOf(value), false)
}

func redactSecrets(curr reflect.Value, redact bool) error {
	if !curr.IsValid() {
		return nil
	}

	actualCurrValue := curr
	if curr.Kind() == reflect.Ptr {
		actualCurrValue = curr.Elem()
	}

	switch actualCurrValue.Kind() {
	case reflect.Map:
		for _, v := range actualCurrValue.MapKeys() {
			err := redactSecrets(actualCurrValue.MapIndex(v), false)
			if err != nil {
				return err
			}
		}
		return nil

	case reflect.String:
		if redact && !actualCurrValue.IsZero() {
			actualCurrValue.SetString(redacted)
		}
		return nil

	case reflect.Slice:
		if actualCurrValue.Type() == reflect.TypeOf([]byte{}) && redact {
			if !actualCurrValue.IsNil() {
				actualCurrValue.SetBytes([]byte(redacted))
			}
			return nil
		}
		for i := range actualCurrValue.Len() {
			err := redactSecrets(actualCurrValue.Index(i), false)
			if err != nil {
				return err
			}
		}
		return nil

	case reflect.Struct:
		for fieldIndex := range actualCurrValue.NumField() {
			currFieldValue := actualCurrValue.Field(fieldIndex)
			currFieldType := actualCurrValue.Type().Field(fieldIndex)
			policyTag := currFieldType.Tag.Get(dataPolicyTag)
			policy := strings.Split(policyTag, ",")[0]

			if policy == "secret" {
				err := redactSecrets(currFieldValue, true)
				if err != nil {
					return err
				}
			} else {
				err := redactSecrets(currFieldValue, false)
				if err != nil {
					return err
				}
			}
		}
		return nil

	default:
		return nil
	}
}
