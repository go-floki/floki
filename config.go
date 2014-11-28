package floki

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"time"
)

var (
	ConfigFile = flag.String("config", "./app/config/%s.json", "Specify application config file to use")
)

type ConfigMap struct {
	data map[string]*json.RawMessage
}

func (f *Floki) loadConfig() {
	logger := f.logger

	f.triggerAppEvent("ConfigureAppStart")

	flag.Parse()

	configFileName := fmt.Sprintf(*ConfigFile, Env)
	if Env == Dev {
		logger.Println("Using config file:", configFileName)
	}

	err := loadConfig(configFileName, &f.Config)
	if err != nil {
		logger.Fatalln("Error loading config file:", configFileName, ":", err)
	}

	// init time zone
	timeZoneStr := f.Config.Str("timeZone", "")

	TimeZone, err = time.LoadLocation(timeZoneStr)
	if err != nil {
		logger.Println("Invalid timezone in configuration file specified:", timeZoneStr, ". Falling back to UTC")
		TimeZone, err = time.LoadLocation("")
	}

	f.triggerAppEvent("ConfigureAppEnd")

	logger.Println("loaded config:", configFileName)
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
		return ConfigMap{make(map[string]*json.RawMessage)}
	}

	var m map[string]*json.RawMessage
	json.Unmarshal(*v, &m)
	return ConfigMap{m}
}

func (c ConfigMap) Keys() []string {
	var keys []string
	for k := range c.data {
		keys = append(keys, k)
	}
	return keys
}

func (c ConfigMap) EachMap(iterator func(key string, value ConfigMap)) {
	for k := range c.data {
		var m map[string]*json.RawMessage
		json.Unmarshal(*c.data[k], &m)
		iterator(k, ConfigMap{m})
	}
}

// loadConfig gets your config from the json file,
// and returns resulting ConfigMap
func loadConfig(filename string, o *ConfigMap) error {
	b, err := ioutil.ReadFile(filename)
	if err == nil {
		err = json.Unmarshal(b, &o.data)
		return err
	}

	return err
}
