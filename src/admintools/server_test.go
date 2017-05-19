package main_test

import (
	"testing"
	"flag"
//	"os"
	"os"
)

var config = flag.String("config", "config.json", "configuration file")

func TestMain(m *testing.M){
	os.Exit(m.Run())
}

func RedirectTest(t *testing.T){

}