// Package skawld is the anti-corruption boundary between the maintenance
// product and skawld-sdk-go. No other product package may import the SDK.
//
// The SDK dependency is intentionally not added until ADR 0002's clean release
// gate is satisfied. Product-owned ports can be introduced here without
// leaking SDK types into maintenance domain or application packages.
package skawld

const IntegrationStatus = "awaiting_compatible_sdk_release"
