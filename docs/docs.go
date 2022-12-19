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
		idpool   = d.Struct(tt.IDPool{})
		inner    = d.Interface((*tt.Inner)(nil))
		outer    = d.Interface((*tt.Outer)(nil))
		listener = d.Struct(tt.Listener{})

		logger   = d.Struct(tt.Logger{})
		qsupport = d.Struct(tt.QualitySupport{})
	)

	d.Place(router).At(100, 100)
	//d.Place(handler).RightOf(router)
	d.Place(idpool, inner, outer).RightOf(router)

	d.Place(listener).Below(idpool)

	d.Place(logger, qsupport).RightOf(listener).Move(100, 0)
	return d
}
