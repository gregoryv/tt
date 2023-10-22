package main

import "testing"

func Test_serveconn(t *testing.T) {
	ServeNewConnection().SaveAs("serveconn.svg")
}
