package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/someara/terraform-provider-zerotier/pkg/zerotier-client"
)

var (
	controllerToken = os.Getenv("ZEROTIER_CONTROLLER_TOKEN")
	controllerURL   = os.Getenv("ZEROTIER_CONTROLLER_URL")
)

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

			controllerToken = strings.TrimSpace(string(content))
		}
	}
	rc, err := filepath.Abs("test.tfrc")
	if err != nil {
		panic(err)
	}

	os.Setenv("TF_CLI_CONFIG_FILE", rc)
	os.Setenv("ZEROTIER_CONTROLLER_TOKEN", controllerToken)

	if controllerURL == "" {
		controllerURL = zerotier.HostURL
		os.Setenv("ZEROTIER_CONTROLLER_URL", zerotier.HostURL)
	}

	os.Exit(m.Run())
}
