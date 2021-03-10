package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/zerotier/go-ztcentral"
)

// given a token, nuke all networks under that token.
func main() {
	controllerToken := os.Getenv("ZEROTIER_CENTRAL_TOKEN")
	if controllerToken == "" {
		if _, err := os.Stat("test-token.txt"); err != nil {
			fmt.Println("Please supply ZEROTIER_CENTRAL_TOKEN in the environment or test-token.txt on disk with the token assigned.")
			panic(err)
		}

		content, err := ioutil.ReadFile("test-token.txt")
		if err != nil {
			panic(err)
		}

		controllerToken = strings.TrimSpace(string(content))
	}

	fmt.Printf("This reaps all networks for the token: %q\n", controllerToken)
	c := ztcentral.NewClient(controllerToken)
	networks, err := c.GetNetworks(context.Background())
	if err != nil {
		panic(err)
	}

	for _, network := range networks {
		fmt.Println(network.Config.Name)
	}

	fmt.Println("These networks will be deleted. Press enter to continue, ^C to cancel")
	buf := make([]byte, 1)
	os.Stdin.Read(buf)

	for _, network := range networks {
		if err := c.DeleteNetwork(context.Background(), network.ID); err != nil {
			fmt.Fprintf(os.Stderr, "Error deleting network: %v; barrelling forward anyway", err)
		}
	}
}
