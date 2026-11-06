package main

import (
	"context"
	"flag"
	"log"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5/tf5server"
	"github.com/hashicorp/terraform-plugin-mux/tf5muxserver"
	"github.com/zerotier/terraform-provider-zerotier/pkg/zerotier"
	"github.com/zerotier/terraform-provider-zerotier/pkg/zerotierv2"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "run the provider with support for debuggers like delve")
	flag.Parse()

	ctx := context.Background()

	server, err := ProviderServer(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var opts []tf5server.ServeOpt
	if debug {
		opts = append(opts, tf5server.WithManagedDebug())
	}

	if err := tf5server.Serve("registry.terraform.io/zerotier/zerotier", server, opts...); err != nil {
		log.Fatal(err)
	}
}

// ProviderServer serves the Central V1 resources (SDKv2) and the Central V2
// resources (plugin framework) as one provider.
func ProviderServer(ctx context.Context) (func() tfprotov5.ProviderServer, error) {
	mux, err := tf5muxserver.NewMuxServer(ctx,
		zerotier.Provider().GRPCProvider,
		providerserver.NewProtocol5(zerotierv2.New(zerotier.Version)()),
	)
	if err != nil {
		return nil, err
	}

	return mux.ProviderServer, nil
}
