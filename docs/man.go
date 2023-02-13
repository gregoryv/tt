// Command feature documents tt features
//
// The command executes each feature and writes the documentation for
// it to stdout.
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	. "github.com/gregoryv/web"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)

	doc := Article(
		H1("tt - manual"),

		Section(
			H2("-h, --help"),
			Pre(
				must(exec.Command("tt", "-hs")),
			),
		),

		Section(H2("Design"),
			NewDesignDiagram().Inline(),
			NewConnectCleanStart().Inline(),
		),
	)

	// compose manual page
	NewFile("man.html",
		Html(
			Body(doc),
		),
	).SaveTo(".")
}

func must(cmd *exec.Cmd) string {
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Output(2, fmt.Sprint(cmd, err))
		os.Exit(1)
	}
	return string(out)
}
