package floki

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type ConfigMap struct {
	data map[string]*json.RawMessage
}

func (c ConfigMap) Bool(key string, defaultValue bool) bool {
	v := c.data[key]

	if v == nil {
		return defaultValue
	}

	var b bool
	json.Unmarshal(*v, &b)
	return b
}

func (c ConfigMap) Int(key string, defaultValue int) int {
	v := c.data[key]

	if v == nil {
		return defaultValue
	}

	var i int
	json.Unmarshal(*v, &i)
	return i
}

func (c ConfigMap) Str(key string, defaultValue string) string {
	v := c.data[key]

	if v == nil {
		return defaultValue
	}

	var s string
	json.Unmarshal(*v, &s)
	return s
}

func (c ConfigMap) Map(key string) ConfigMap {
	v := c.data[key]

	if v == nil {
		return nil
	}

	var m map[string]*json.RawMessage
	json.Unmarshal(*v, &m)
	return ConfigMap{m}
}

// loadConfig gets your config from the json file,
// and returns resulting ConfigMap
func loadConfig(filename string, o *ConfigMap) error {
	b, err := ioutil.ReadFile(filename)
	if err == nil {
		err = json.Unmarshal(b, &o.data)

		fmt.Println(o)

		return err
	}

	return err
}
