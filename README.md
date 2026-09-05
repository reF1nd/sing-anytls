# sing-anytls (reF1nd)

This fork retains SagerNet's rewritten AnyTLS implementation and session lifecycle
APIs, with compatibility for the features previously used by reF1nd/sing-box.

## Client metadata

`ClientOptions.ClientMetadata` is a `*string`:

- `nil`: send `sing-anytls/0.0.13`.
- Pointer to `""`: send an empty `client=` setting, without client identification.
- Pointer to a custom string: send that string unchanged.

The default is a legacy compatibility identity, **not** the version of this
rewritten library. sing-box appends ` sing-box/<version>` to its default value.
Callers migrating from SagerNet's string field must pass a pointer for an explicit
value. The constructor copies the value; it does not retain the pointer.

## Compatibility coverage

The existing implementation already includes the protocol-v2 features of the old
library through 0.0.13: padding updates, large-write frame splitting, disabling
session reuse, and sending FIN before releasing/closing the session. Version
0.0.12 added the reuse/close changes; 0.0.13 corrected the legacy version identifier.

- `DisableReuse` closes the session after its stream closes, even when idle
  retention or `ContextWithKeepSession` is requested.
- `Reset` closes active and idle sessions without permanently closing the client.
- `SetKeepIdleConnections`, `CloseIdleConnections`, and `ContextWithKeepSession`
  retain SagerNet's on-demand idle-connection controls.
- `ServiceOptions.FallbackHandler` works with both single-user and multi-user
  services and receives the original, unconsumed authentication-failure data.
  ALPN destination selection belongs to sing-box's TLS/inbound layer. As in the
  existing implementation, a first read shorter than the password is sent to
  fallback; authenticated padding read failures close the connection.
- A dialer whose addresses are unavailable before its first write is supported.
  TCP Fast Open socket setup and TLS negotiation belong to the caller; sing-box
  tests exercise that full path separately.

The nested `test` module runs bidirectional interoperability against the official
`github.com/anytls/sing-anytls v0.0.13`, as well as metadata wire, fallback byte
preservation, reuse/idle-policy, reset, lazy-address, and existing stress tests.

```sh
go test -race ./...
cd test
go test -race -timeout 3m ./...
```
