package main

import (
	"os/exec"
	"testing"
)

func TestNew(t *testing.T) {
	c := New()
	cmd := Command{Command: "pwd", Timeout: 2}
	c.AddCommand(cmd)
	exe, err := exec.Command("pwd").Output()
	if err != nil {
		t.Error("Error in exec.Command function")
	}
	if string(exe) != c.RunCommands() {
		t.Error("RunCommands is not working.")
	}
}

func TestRunCommand(t *testing.T) {
	//cmd := Command{Command:"ls", Timeout:3}
	//exe, err := exec.Command("ls").Output()
	//if err != nil {
	//	t.Error("Error in exec.Command function")
	//}
	//var out, oerr bytes.Buffer
	//if cmd.Run(&out, &oerr) != string(exe) {
	//	t.Error("Run is not working")
	//}
	//t.Log(out.String(), oerr.String())
}
