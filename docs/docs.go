package main

import (
	"github.com/gregoryv/draw/design"
	"github.com/gregoryv/tt"
)

func NewDesignDiagram() *design.ClassDiagram {
	var (
		d = design.NewClassDiagram()
		//handler = d.Interface((*tt.Handler)(nil)) // func, unsupported in draw/design :-/

		recv = d.Struct(tt.Receiver{})
		remote   = d.Interface((*tt.Connection)(nil))

		server = d.Struct(tt.Server{})

		_ = []design.VRecord{
			recv, remote, server,
		}
	)
	d.Style.Spacing = 70
	d.HideRealizations()

	d.Place(server).At(120, 20)
	return d
}
