package docs

import (
	"github.com/gregoryv/draw/design"
	"github.com/gregoryv/tt"
)

func NewDesignDiagram() *design.ClassDiagram {
	var (
		d      = design.NewClassDiagram()
		router = d.Struct(tt.Router{})
		//handler = d.Interface((*tt.Handler)(nil)) // func, unsupported in draw/design :-/
		listener = d.Struct(tt.Listener{})

		receiver = d.Struct(tt.Receiver{})
		remote   = d.Interface((*tt.Remote)(nil))

		server = d.Struct(tt.Server{})

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
