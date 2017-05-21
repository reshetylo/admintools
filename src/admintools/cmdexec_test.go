package main

import (
	"os/exec"
	"testing"
)

func TestNew(t *testing.T) {
	c := New()
	cmd := Command{Command: "pwd", Timeout: 3}
	c.AddCommand(cmd)
	exe, _ := exec.Command("pwd").Output()
	t.Log(string(exe))
	if string(exe) != c.RunCommands() {
		t.Error("RunCommands is not working.")
	}
}
