package utils

import (
	"encoding/json"
	"fmt"
)

// ToJSON 将对象转换为 JSON 字符串
func ToJSON(v interface{}) string {
	bytes, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("json marshal error: %v", err)
	}
	return string(bytes)
}

// ToPrettyJSON 将对象转换为格式化的 JSON 字符串
func ToPrettyJSON(v interface{}) string {
	bytes, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("json marshal error: %v", err)
	}
	return string(bytes)
}

// FromJSON 将 JSON 字符串解析为对象
func FromJSON(data string, v interface{}) error {
	return json.Unmarshal([]byte(data), v)
}
