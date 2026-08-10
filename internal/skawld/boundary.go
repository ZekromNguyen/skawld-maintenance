// Package skawld is the anti-corruption boundary between the maintenance
// product and skawld-sdk-go. No other product package may import the SDK.
//
// Product-owned ports remain the maintenance-facing contracts. SDK types are
// translated here and never leak into maintenance domain/application models.
package skawld

const (
	SDKVersion        = "v0.2.0"
	IntegrationStatus = "pinned_contracts_verified"
)
