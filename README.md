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
