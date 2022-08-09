package model

import (
	"encoding/json"
)

func FixS3Bucket(data []byte) ([]byte, error) {
	var m map[string]interface{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return nil, err
	}
	if desc, ok := m["Description"]; ok {
		d := desc.(map[string]interface{})
		if v, ok := d["LifecycleRules"]; ok {
			rules := v.([]map[string]interface{})
			for i, rule := range rules {
				if _, ok := rule["Filter"]; ok {
					delete(rule, "Filter")
					rules[i] = rule
				}
			}
			d["LifecycleRules"] = rules
		}
		m["Description"] = d
	}
	return json.Marshal(m)
}
