// Package ctypb aims to provide a bidirectional bridge between comparable
// cty and protocol buffers concepts.
//
// "cty" (see-tie) is, in a sense, a reflection API for a language that
// doesn't exist. It aims to provide a dynamic type system which preserves
// a set of useful invariants that calling applications can rely on when
// using cty as a building-block for data whose structure isn't predictable
// at compile time.
//
// Protocol buffers is a serialization format and associated schema language
// intended for transmitting messages between systems, often in an RPC style.
// Protocol buffers typically encourages a static definition of types written
// in a schema, though it is in principle possible to also define a schema
// at runtime. Either way, protocol buffers messages are not self-describing
// so a schema is required to parse one.
package ctypb
