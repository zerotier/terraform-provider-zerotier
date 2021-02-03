package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

var controllerToken = os.Getenv("ZEROTIER_CONTROLLER_TOKEN")

func TestMain(m *testing.M) {
	if controllerToken == "" {
		if fi, err := os.Stat("test-token.txt"); err != nil {
			fmt.Println("test-token.txt not present in tree; ZEROTIER_CONTROLLER_TOKEN is required in environment for many tests.")
		} else if fi.Mode()&os.ModeIrregular != 0 {
			panic("test-token.txt is not a regular file; not sure what to do here, so bailing")
		} else {
			content, err := ioutil.ReadFile("test-token.txt")
			if err != nil {
				panic(err)
			}

			controllerToken = string(content)
		}
	}

	os.Setenv("ZEROTIER_CONTROLLER_TOKEN", controllerToken)
	os.Exit(m.Run())
}
