package main

import (
	"github.com/gregoryv/draw/design"
	"github.com/gregoryv/tt"
	"github.com/gregoryv/tt/ttsrv"
)

func NewDesignDiagram() *design.ClassDiagram {
	var (
		d = design.NewClassDiagram()
		//handler = d.Interface((*tt.Handler)(nil)) // func, unsupported in draw/design :-/

		receiver = d.Struct(tt.Receiver{})
		remote   = d.Interface((*ttsrv.Connection)(nil))

		server = d.Struct(ttsrv.Server{})

		_ = []design.VRecord{
			receiver, remote, server,
		}
	)
	d.Style.Spacing = 70
	d.HideRealizations()

	d.Place(server).At(120, 20)
	return d
}
