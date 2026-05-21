// Package connector contains the mautrix bridgev2 network connector for Snapchat.
//
// The package is intentionally shaped like other modern mautrix bridge
// connectors: this layer owns login flows, bridge metadata, portal/ghost/message
// conversion, polling, and Matrix send handling. The browser sidecar remains an
// internal transport detail and is imported as sidecar where needed.
package connector
