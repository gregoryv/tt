[gregoryv/tt](https://pkg.go.dev/github.com/gregoryv/tt) - package provides components for writing [mqtt-v5.0](https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html) clients and servers

<img src="./etc/logo.svg" />

## Quick start

    $ go install github.com/gregoryv/tt/cmd/tt@latest
    $ tt -h

Even though this repository provides a basic client/server(WIP) command
it's main purpose is to provide capabilities for others to write their
own clients depending on their situation.

This package uses the sibling package
[gregoryv/mq](https://github.com/gregoryv/mq) for encoding control
packets on the wire.

The design focuses on decoupling each specific feature as much as
possible.  One example being the creation of a network connection is
not enforced, which makes it easy to create inmemory clients and
servers during testing.
