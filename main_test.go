package main

import (
	"fmt"
	"os/exec"
	"testing"
)

func Test_runPipe(t *testing.T) {
	cmd := "ls -l | grep main.go"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		fmt.Printf("Failed to execute command: %s", cmd)
	}
	ss := string(out)
	fmt.Println(ss)
}

func Test_findMaxCapacity(t *testing.T) {
	findMaxCapacity("/Users/qa/go/", "aassdd")
}
