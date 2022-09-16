package config

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
)

func ReadFromEnv(configObj interface{}, prefix []string) {
	reflectType := reflect.TypeOf(configObj).Elem()
	reflectValue := reflect.ValueOf(configObj).Elem()

	for i := 0; i < reflectType.NumField(); i++ {
		fieldType := reflectType.Field(i)
		typeName := fieldType.Name
		yamlName := fieldType.Tag.Get("yaml")
		if yamlName == "" {
			yamlName = typeName
		}
		yamlName = strings.ToUpper(yamlName)

		fieldValue := reflectValue.Field(i)
		valueType := fieldValue.Type()
		valueValue := fieldValue.Addr()

		switch fieldValue.Kind() {
		case reflect.String:
			v := getEnv(prefix, yamlName)
			fieldValue.SetString(v)
		case reflect.Int, reflect.Int32, reflect.Int64:
			v, err := strconv.ParseInt(getEnv(prefix, yamlName), 10, 64)
			if err != nil {
				panic(err)
			}
			fieldValue.SetInt(v)
		case reflect.Bool:
			v, err := strconv.ParseBool(getEnv(prefix, yamlName))
			if err != nil {
				panic(err)
			}
			fieldValue.SetBool(v)
		case reflect.Struct:
			ReadFromEnv(valueValue.Interface(), append(prefix, yamlName))
		default:
			panic(fmt.Errorf("%s : it is %s\n", typeName, valueType))
		}
	}
}

func getEnv(prefix []string, key string) string {
	return os.Getenv(strings.Join(append(prefix, key), "_"))
}
