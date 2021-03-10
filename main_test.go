package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erikh/tftest"
	"github.com/zerotier/go-ztcentral"
)

var (
	controllerToken = os.Getenv("ZEROTIER_CENTRAL_TOKEN")
	controllerURL   = os.Getenv("ZEROTIER_CENTRAL_URL")
)

func TestMain(m *testing.M) {
	if controllerToken == "" {
		if fi, err := os.Stat("test-token.txt"); err != nil {
			fmt.Println("test-token.txt not present in tree; ZEROTIER_CENTRAL_TOKEN is required in environment for many tests.")
		} else if fi.Mode()&os.ModeIrregular != 0 {
			panic("test-token.txt is not a regular file; not sure what to do here, so bailing")
		} else {
			content, err := ioutil.ReadFile("test-token.txt")
			if err != nil {
				panic(err)
			}

			controllerToken = strings.TrimSpace(string(content))
		}
	}
	rc, err := filepath.Abs("test.tfrc")
	if err != nil {
		panic(err)
	}

	os.Setenv("TF_CLI_CONFIG_FILE", rc)
	os.Setenv("ZEROTIER_CENTRAL_TOKEN", controllerToken)

	if controllerURL == "" {
		controllerURL = ztcentral.BaseURLV1
		os.Setenv("ZEROTIER_CENTRAL_URL", controllerURL)
	}

	os.Exit(m.Run())
}

func getTFTest(t *testing.T) *tftest.Harness {
	tf := tftest.New(t)
	tf.HandleSignals(true)
	return tf
}
