# Changelog

This project adheres to semantic versioning and all notable
changes will be documented in this file.

## [unreleased]

- Add type ttsrv.Logger with network and clientID in log lines

## [0.8.0] 2023-02-11

- Reciver.Run can is stopped by StopReceiver
- Rename type ttsrv.Remote -> Connection
- Move type Remote to ttsrv
- Rename type FilterExpr -> TopicFilter
- Move type ClosedConn to ttx
- Move tt.Start to cmd/tt
- Add type ttsrv/IDPool
- Move server related components to subdirectory ttsrv
- Replace type MemConn with testnet.Conn
- Ignore Disconnect in pubcmd

## [0.7.0] 2023-01-15

- Use latest gregoryv/mq
- Hide field Server.Router
- Hide field Server.Logger
- Hide field Router.Logger
- Add docs directory with design digram

## [0.6.0] 2022-12-17

- Use gregoryv/mq v0.26.0
- Add type FilterExpr
- Don't log read errors from closed connections
- Server disconnects on incomming malformed publish
- Add type Listener
- Decouple server from net.Listener
- Add server statistics

## [0.5.0] 2022-11-30

- tt pub exits when message is published successfully
- Handle interruptions gracefully
- Subscriber responds with SubAck
- Reverse order of CombineIn and CombineOut
- Use gregoryv/mq@902e660
- Use Router in Server
- Add type ClientIDMaker
- Server supports QoS 0 only with proper disconnect semantics
- Add type QualitySupport
- Add field Server.Router
- Add type Remote interface
- Logger logs optional remote addr on mq.Connect

## [0.4.0] 2022-11-25

- Use pink gopher in logo

## [0.3.0] 2022-11-24

- Add flags --debug and --client-id to cmd/tt sub 
- Add flags --username and --password to cmd/tt pub
- Remove type LogLevel
- Add types PubHandler, Transmitter

## [0.2.0] 2022-11-06

- Add type TCP
- Remove types Sender and SubWait
- Add types Handler and Middleware
- Rename InFlow and OutFlow to Inner and Outer
- Move type Server to root
- Remove funcs NewInQueue and NewOutQueue
- Fix imports of mq/tt
- Use gregoryv/mq@v0.22.0

## [0.1.0] 2022-11-06

- Moved as separate go module from mq/v0.21.0
