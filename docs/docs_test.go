package docs

import "testing"

func Test_generateDiagram(t *testing.T) {
	if err := NewDesignDiagram().SaveAs("design.svg"); err != nil {
		t.Fatal(err)
	}
	_ = NewConnectCleanStart().SaveAs("connect_clean_start.svg")
}
