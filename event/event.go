// Package event provides client and server event types.
//
// In addition to packet types event types are used to inform the
// application of what is happening inside a [tt.Client] or
// [tt.Server].
//
// [tt.Client]: https://pkg.go.dev/github.com/gregoryv/tt#Client
// [tt.Server]: https://pkg.go.dev/github.com/gregoryv/tt#Server
package event

// ClientUp indicates client is ready for sending packets.
type ClientUp uint8
type ClientConnect uint8
type ClientConnectFail string
type ClientStop uint8

type ServerUp uint8
type ServerDown uint8
