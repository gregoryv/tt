package main

import (
	"net"

	"github.com/gregoryv/draw/design"
	"github.com/gregoryv/tt"
)

func ServeNewConnection() *design.SequenceDiagram {
	d := design.NewSequenceDiagram()
	d.ColWidth = 120
	var (
		srv   = d.AddStruct(tt.Server{})
		ln    = d.AddInterface((*net.Listener)(nil))
		cfeed = d.Add("tt.connFeed")
		rec   = d.Add("tt.receiver")
	)
	d.Link(srv, srv, "once.setDefaults")
	d.Link(srv, srv, "go run")
	d.Link(srv, ln, "new : ln")
	d.Link(srv, cfeed, "new(ln)")
	d.Link(srv, cfeed, "Run(ctx)")
	d.Link(srv, rec, "...")
	d.SetCaption("Figure 1. Showing tt.Server.Start(ctx)")
	return d
}
