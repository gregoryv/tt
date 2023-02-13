package main

import (
	"github.com/gregoryv/draw/design"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/ttsrv"
)

func NewDesignDiagram() *design.ClassDiagram {
	var (
		d      = design.NewClassDiagram()
		router = d.Struct(ttsrv.Router{})
		//handler = d.Interface((*tt.Handler)(nil)) // func, unsupported in draw/design :-/
		listener = d.Struct(ttsrv.Listener{})

		receiver = d.Struct(tt.Receiver{})
		remote   = d.Interface((*ttsrv.Connection)(nil))

		server = d.Struct(ttsrv.Server{})

		_ = []design.VRecord{
			router, listener,
			receiver, remote, server,
		}
	)
	d.Style.Spacing = 70
	d.HideRealizations()

	d.Place(server).At(120, 20)
	d.Place(router).Below(server)
	return d
}
