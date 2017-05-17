package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"time"

	"gopkg.in/yaml.v2"
)

type fileFormat struct {
	Name           string
	Version        string
	DefaultTimeout int       "default_timeout"
	Commands       []Command "commands"
}

type fileCache map[string]struct {
	file fileFormat
	time int64
}

var filecache = make(fileCache, 100)

func readFile(file string) fileFormat {
	var filedata fileFormat
	filedata, err := getCache(file)
	if err != nil {
		// cache does not exist. read config file
		source, err := ioutil.ReadFile(file)
		if err != nil {
			panic(err)
		}
		err = parseYAML(source, &filedata)
		if err != nil {
			panic(err)
		}
		saveCache(file, filedata)
	}
	return filedata
}

func getCache(file string) (fileFormat, error) {
	if filecache[file].time <= time.Now().Unix()-fileCacheTime {
		return fileFormat{}, errors.New("Cache expired")
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
