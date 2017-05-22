package main

import (
	"encoding/json"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

type fileFormat struct {
	Name           string    `json:"name" yaml:"name"`
	Version        string    `json:"version" yaml:"version"`
	DefaultTimeout int       `json:"default_timeout" yaml:"default_timeout"`
	Commands       []Command `json:"commands" yaml:"commands"`
}

type fileCache map[string]struct {
	file fileFormat
	time int64
}

var filecache = make(fileCache, 100)

func readFile(file string) (filedata fileFormat, err error) {
	filedata, err = getCache(file)
	if err != nil {
		// cache does not exist. read config file
		source, err := ioutil.ReadFile(file)
		if err != nil {
			return filedata, errorNew("Can not read file:", file, err.Error())
		}

		format := file[len(file)-4:]
		if format == "yaml" {
			err = parseYAML(source, &filedata)
		} else if format == "json" {
			err = parseJSON(source, &filedata)
		} else {
			return filedata, errorNew("Do not support file format:", format)
		}

		if err != nil {
			return filedata, errorNew("Can not parse file:", file, err.Error())
		}
		saveCache(file, filedata)
	}
	return filedata, nil
}

func getCache(file string) (fileFormat, error) {
	if filecache[file].time <= time.Now().Unix()-fileCacheTime {
		return fileFormat{}, errorNew("Cache expired")
	} else {
		cacheData := filecache[file].file
		commands := make([]Command, len(filecache[file].file.Commands))
		copy(commands, filecache[file].file.Commands)
		cacheData.Commands = commands
		return cacheData, nil
	}
}

func saveCache(file string, filedata fileFormat) {
	var tmp = filecache[file]
	tmp.file = filedata
	commands := make([]Command, len(filedata.Commands))
	copy(commands, filedata.Commands)
	tmp.file.Commands = commands
	tmp.time = time.Now().Unix()
	filecache[file] = tmp
}

func parseYAML(source []byte, output interface{}) (err error) {
	return yaml.Unmarshal(source, output)
}

func parseJSON(source []byte, output interface{}) (err error) {
	return json.Unmarshal(source, output)
}
