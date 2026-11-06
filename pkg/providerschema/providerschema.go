// Package providerschema holds the provider configuration attributes shared by
// the Central V1 (SDKv2) and Central V2 (plugin framework) halves of the
// provider. terraform-plugin-mux requires both halves to declare an identical
// provider schema, so names and descriptions live here and nowhere else.
package providerschema

const (
	CentralURL      = "zerotier_central_url"
	CentralToken    = "zerotier_central_token"
	CentralV2URL    = "zerotier_central_v2_url"
	CentralV2APIKey = "zerotier_central_v2_api_key"
)

const (
	CentralURLDescription      = "ZeroTier Central V1 API endpoint. Unlikely you'll need to alter this unless you're testing ZeroTier central itself. Can also be set with ZEROTIER_CENTRAL_URL."
	CentralTokenDescription    = "ZeroTier Central V1 API Token; you can generate a new one at https://my.zerotier.com/account. Required for the V1 resources (zerotier_network, zerotier_member, zerotier_token). Can also be set with ZEROTIER_CENTRAL_TOKEN."
	CentralV2URLDescription    = "ZeroTier Central V2 API endpoint. Defaults to https://central.zerotier.com. Can also be set with ZEROTIER_CENTRAL_V2_URL."
	CentralV2APIKeyDescription = "ZeroTier Central V2 service account API key. Required for the zerotier_v2_* resources. Can also be set with ZEROTIER_CENTRAL_V2_API_KEY."
)
