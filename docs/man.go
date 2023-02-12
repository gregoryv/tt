// Command feature documents tt features
//
// The command executes each feature and writes the documentation for
// it to stdout.
package main

import (
	"log"
	"os"
	"os/exec"

	. "github.com/gregoryv/web"
)

func main() {

	doc := Article(
		H1("tt - manual"),

		Section(
			H2("-h, --help"),
			Pre(
				must(exec.Command("tt", "-h")),
			),
		),

		Section(H2("Design"),
			NewDesignDiagram().Inline(),
			NewConnectCleanStart().Inline(),
		),
	)

	// compose manual page
	NewPage(
		Html(
			Body(doc),
		),
	).WriteTo(os.Stdout)
}

func must(cmd *exec.Cmd) string {
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(cmd, err)
	}
	return string(out)
}
