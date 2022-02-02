package describer

import (
	"encoding/json"
	"reflect"
	"strings"
)

// JSONAllFieldsMarshaller is a hack around the issue described here
// https://githubmemory.com/repo/Azure/azure-sdk-for-go/issues/12227
// Azure sdk overrides all the MarshalJSON methods for the struct fields
// to exclude the 'READ-ONLY' fields from the JSON output of the struct.
// By simply wrapping the original struct by JSONAllFieldsMarshaller, all
// the fields will appear in the json output.
type JSONAllFieldsMarshaller struct {
	Value interface{}
}

func (x JSONAllFieldsMarshaller) MarshalJSON() ([]byte, error) {
	var val interface{} = x.Value

	v := reflect.ValueOf(x.Value)
	if isAzureType(v.Type()) {
		switch v.Kind() {
		case reflect.Slice, reflect.Array:
			if isAzureType(v.Type().Elem()) {
				val = azSliceMarshaller{Value: v}
			}
		case reflect.Ptr:
			if isAzureType(v.Type().Elem()) {
				val = azPtrMarshaller{Value: v}
			}
		case reflect.Struct:
			val = azStructMarshaller{Value: v}
		}
	}

	return json.Marshal(val)
}

type azStructMarshaller struct {
	reflect.Value
}

func (x azStructMarshaller) MarshalJSON() ([]byte, error) {
	v := x.Value
	m := make(map[string]interface{})
	num := v.Type().NumField()
	for i := 0; i < num; i++ {
		field := v.Type().Field(i)
		jsonTag := field.Tag.Get("json")
		jsonFields := strings.Split(jsonTag, ",")
		jsonField := jsonFields[0]
		if jsonField == "-" {
			continue
		} else if jsonField == "" {
			jsonField = field.Name
		}

		jsonOmitEmpty := false
		for _, field := range jsonFields {
			if field == "omitempty" {
				jsonOmitEmpty = true
				break
			}
		}
		if jsonOmitEmpty && isEmptyValue(v.Field(i)) {
			continue
		}

		m[jsonField] = JSONAllFieldsMarshaller{Value: v.Field(i).Interface()}
	}

	return json.Marshal(m)
}

type azPtrMarshaller struct {
	reflect.Value
}

func (x azPtrMarshaller) MarshalJSON() ([]byte, error) {
	val := x.Value
	for val.Type().Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	return JSONAllFieldsMarshaller{Value: val.Interface()}.MarshalJSON()
}

type azSliceMarshaller struct {
	reflect.Value
}

func (x azSliceMarshaller) MarshalJSON() ([]byte, error) {
	num := x.Value.Len()
	list := make([]JSONAllFieldsMarshaller, 0, num)
	for i := 0; i < num; i++ {
		list = append(list, JSONAllFieldsMarshaller{Value: x.Value.Index(i).Interface()})
	}

	return json.Marshal(list)
}

func isAzureType(t reflect.Type) bool {
	return strings.HasPrefix(t.PkgPath(), "github.com/Azure/azure-sdk-for-go") ||
		// TODO: Wrap the objects before putting them into model. That way we don't have to include the second package here
		strings.HasPrefix(t.PkgPath(), "gitlab.com/keibiengine/keibi-engine/pkg/azure/model")
}

func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr:
		return v.IsNil()
	}
	return false
}
