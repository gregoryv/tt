package docs

import "testing"

func TestDesignDiagram(t *testing.T) {
	d := NewDesignDiagram()
	d.SaveAs("design.svg")
}
