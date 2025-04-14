package config

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

func SetValue[V any](input *V, path string, value string) error {
	pathParts := strings.Split(path, ".")

	return updateValue(reflect.ValueOf(input), pathParts, value, false)
}

func UnsetValue[V any](input *V, path string) error {
	pathParts := strings.Split(path, ".")

	return updateValue(reflect.ValueOf(input), pathParts, "", true)
}

//nolint:gocyclo
func updateValue(input reflect.Value, path []string, value string, unset bool) error {
	// Just don't want to deal with pointers later.
	actualInput := input
	if input.Kind() == reflect.Pointer {
		actualInput = input.Elem()
	}

	// TODO: handle more Kinds as needed by the config structs
	switch actualInput.Kind() {
	case reflect.Struct:
		if len(path) == 0 && unset {
			actualInput.Set(reflect.New(actualInput.Type()).Elem())
			return nil
		}

		if len(path) == 0 {
			return errors.New("can not set struct")
		}

		step := path[0]
		path = path[1:]

		for fieldIndex := range actualInput.NumField() {
			field := actualInput.Field(fieldIndex)
			fieldType := actualInput.Type().Field(fieldIndex)
			yamlName := strings.Split(fieldType.Tag.Get("yaml"), ",")[0]

			if yamlName != step {
				continue
			}

			if len(path) == 0 && unset {
				newValue := reflect.New(field.Type()).Elem()
				field.Set(newValue)
				return nil
			}

			if field.Kind() == reflect.Map && field.IsNil() {
				newValue := reflect.MakeMap(field.Type())
				field.Set(newValue)
			}

			if field.Kind() == reflect.Ptr && field.IsNil() {
				newValue := reflect.New(field.Type().Elem())
				field.Set(newValue)
			}

			return updateValue(field, path, value, unset)
		}

		return fmt.Errorf("unable to locate path %#v under %s", step, actualInput.Type())
	case reflect.Map:
		if len(path) == 0 {
			return errors.New("can not set map")
		}

		step := path[0]
		path = path[1:]

		mapKey := reflect.ValueOf(step)
		currMapValue := actualInput.MapIndex(mapKey)

		if len(path) == 0 && unset {
			actualInput.SetMapIndex(mapKey, reflect.Value{})
			return nil
		}

		mapEntryDoesNotExist := currMapValue.Kind() == reflect.Invalid
		if mapEntryDoesNotExist {
			currMapValue = reflect.New(actualInput.Type().Elem().Elem()).Elem().Addr()
			actualInput.SetMapIndex(mapKey, currMapValue)
		}

		return updateValue(currMapValue, path, value, unset)
	case reflect.String:
		if len(path) != 0 {
			return fmt.Errorf("more steps after string: %s", strings.Join(path, "."))
		}

		if unset {
			actualInput.SetString("")
			return nil
		}

		actualInput.SetString(value)
	case reflect.Slice:
		if len(path) != 0 {
			return fmt.Errorf("more steps after slice: %s", strings.Join(path, "."))
		}

		if unset {
			actualInput.SetBytes(nil)
			return nil
		}

		actualInput.SetBytes([]byte(value))
	case reflect.Bool:
		if len(path) != 0 {
			return fmt.Errorf("more steps after bool: %s", strings.Join(path, "."))
		}

		if unset {
			actualInput.SetBool(false)
			return nil
		}

		boolValue, err := toBool(value)
		if err != nil {
			return err
		}

		actualInput.SetBool(boolValue)
	case reflect.Int64:
		if len(path) != 0 {
			return fmt.Errorf("more steps after int64: %s", strings.Join(path, "."))
		}

		if unset {
			actualInput.SetInt(0)
			return nil
		}

		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("can not parse value as int64: %s", value)
		}

		actualInput.SetInt(intValue)
	default:
		return fmt.Errorf("unhandled kind %v", actualInput.Kind())
	}

	return nil
}

func toBool(value string) (bool, error) {
	if value == "" {
		return false, nil
	}

	return strconv.ParseBool(value)
}
