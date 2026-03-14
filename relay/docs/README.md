# RELAY Client Documentation

Detailed documentation for the SignalWire RELAY client lives alongside the Go source code in the `pkg/relay/docs/` directory.

## Available Guides

- [Getting Started](../../pkg/relay/docs/getting-started.md) -- installation, environment variables, first call handler
- [Call Methods Reference](../../pkg/relay/docs/call-methods.md) -- every method on the `Call` object with signatures and examples
- [Events](../../pkg/relay/docs/events.md) -- event types, `RelayEvent` struct, call state transitions
- [Messaging](../../pkg/relay/docs/messaging.md) -- sending and receiving SMS/MMS with the RELAY client
- [Client Reference](../../pkg/relay/docs/client-reference.md) -- `NewRelayClient` options, `Run()`, `Stop()`, `Dial()`, reconnect behavior

## Quick Links

- [Package source code](../../pkg/relay/) -- the Go implementation
- [Examples](../examples/) -- runnable example programs
- [RELAY Implementation Guide](../../pkg/relay/RELAY_IMPLEMENTATION_GUIDE.md) -- internal architecture notes
