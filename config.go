package floki

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// Load gets your config from the json file,
// and fills your struct with the option
func loadConfig(filename string, o *map[string]*json.RawMessage) error {
	b, err := ioutil.ReadFile(filename)
	if err == nil {
		err = json.Unmarshal(b, &o)

		fmt.Println(o)

		return err
	}

	return err
}

func boolValue(v *json.RawMessage) bool {
	if v == nil {
		return false
	}

	var b bool
	json.Unmarshal(*v, &b)
	return b
}

func stringValue(v *json.RawMessage) string {
	if v == nil {
		return ""
	}

	var b string
	json.Unmarshal(*v, &b)

	return b
}
