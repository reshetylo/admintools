package main

import (
	"io/ioutil"
	"reflect"
	"strings"
	"testing"
)

const (
	testFileYAML = "../../modules/test/test.yaml"
	testFileJSON = "../../modules/test/test.json"
	testFileBAD  = "../../modules/test/test_bad.yaml"
)

func TestGetCache(t *testing.T) {
	// check when cache is empty
	filedata, err := getCache(testFileYAML)
	if err != nil {
		if err.Error() != "Cache expired" {
			t.Error(err)
		}
	}

	// readfile will save data to cache. compare cache with returned data
	internalFileData, _ := readFile(testFileYAML)
	filedata, _ = getCache(testFileYAML)
	if !reflect.DeepEqual(filedata, internalFileData) {
		t.Error("Cached data is not equal to original file: ", filedata, internalFileData)
	}
}

func TestSaveCache(t *testing.T) {
	internalFileData := internalReadFile(t, testFileJSON, "json")

	// save data straight to cache property
	saveCache(testFileJSON, internalFileData)

	// retreive data and perform couple comparisons
	filedata, err := getCache(testFileJSON)
	if err != nil {
		t.Error("Data was not saved to cache: ", testFileJSON, internalFileData)
	}
	if !reflect.DeepEqual(filedata, internalFileData) {
		t.Error("Cached data corrupted: ", filedata, internalFileData)
	}
}

func TestReadFile(t *testing.T) {
	// compare couple files with internalReader
	compareFiles(t, testFileYAML, "yaml")
	compareFiles(t, testFileJSON, "json")

	// check bad file
	_, err := readFile(testFileBAD)
	if !strings.Contains(err.Error(), "found character that cannot start any token") {
		t.Error("Wrong error received while was checking bad file: ", err)
	}

	// compare json and yaml files with identical configuration
	filedata, erryaml := readFile(testFileYAML)
	if erryaml != nil {
		t.Error("Read file error: ", erryaml)
	}
	filedatajson, errjson := readFile(testFileJSON)
	if errjson != nil {
		t.Error("Read file error: ", errjson)
	}
	if !reflect.DeepEqual(filedata, filedatajson) {
		t.Error("JSON and YAML files are not equal", filedata, filedatajson)
	}
}

// This function compares results from readFile with internalReadFile
func compareFiles(t *testing.T, file string, ftype string) {
	filedata, err := readFile(file)
	if err != nil {
		t.Errorf("Can not read file %v. Error: ", file, err)
	}
	internalFileData := internalReadFile(t, file, ftype)
	if !reflect.DeepEqual(filedata, internalFileData) {
		t.Errorf("File compare results are not the same. %v is not as %v", filedata, internalFileData)
	}
}

// Implementation of internal file reader
func internalReadFile(t *testing.T, file string, ftype string) fileFormat {
	var filedata fileFormat
	source, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal("Can not read file ", file, err)
	}
	if ftype == "yaml" {
		err = parseYAML(source, &filedata)
	} else {
		err = parseJSON(source, &filedata)
	}
	if err != nil {
		t.Fatalf("Can not parse %v file: %v. Error: %v", ftype, file, err)
	}
	return filedata
}
