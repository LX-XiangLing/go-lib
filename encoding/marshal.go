package encoding

import (
	"encoding/json"
)

func JsonEncode(v interface{}) string {
	buf, _ := json.Marshal(v)
	return string(buf)
}

func JsonDecode(s string, v interface{}) error {
	return json.Unmarshal([]byte(s), v)
}

func JsonPretty(v interface{}) string {
	buf, _ := json.MarshalIndent(v, "", "    ")
	return string(buf)
}
