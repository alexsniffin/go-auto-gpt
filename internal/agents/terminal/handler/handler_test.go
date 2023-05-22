package handler

import (
	"fmt"
	"testing"
)

func Test_executeCommand(t *testing.T) {
	s, err := executeCommand("apt-get install python -y", "test")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(s)
}
