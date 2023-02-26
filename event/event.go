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

// ClientConnect indicates a successful ConnAck
type ClientConnect uint8

type ClientConnectFail string

// ClientStop indicates client has stopped
type ClientStop struct {
	Err error
}

// ServerUp indicates server is listening for
type ServerUp uint8

// ServerStop indicates server has stopped
type ServerStop struct {
	Err error
}
