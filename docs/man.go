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
	"time"

	"github.com/gregoryv/draw/design"
	"github.com/gregoryv/mq"
	"github.com/gregoryv/web"
	. "github.com/gregoryv/web"
	"github.com/gregoryv/web/theme"
	"github.com/gregoryv/web/toc"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)

	nav := Nav()

	doc := Article(
		H1("tt - manual"),
		time.Now(),
		Br(),
		nav,

		H2("Options"),

		H3("-h, --help"),
		Pre(
			must(exec.Command("tt", "-h")),
		),

		H2("Commands"),

		H3("pub"),

		H4("Publish QoS 0"),

		// diagram showing package flow, maybe just dump the output
		func() *design.SequenceDiagram {
			var (
				d = design.NewSequenceDiagram()
				c = d.Add("client")
				s = d.Add("server")
			)
			d.ColWidth = 300
			{
				p := mq.NewConnect()
				p.SetCleanStart(true)
				d.Link(c, s, p.String())
			}
			{
				p := mq.NewConnAck()
				p.SetTopicAliasMax(10) // default in mosquitto
				d.Link(s, c, p.String())
			}
			d.Link(c, s, mq.Pub(0, "gopher/pink", "hug").String())
			d.Link(c, s, mq.NewDisconnect().String())

			d.SetCaption("Client connects with clean start flag set to true")
			return d
		}().Inline(),

		Pre(must(exec.Command("tt"))),

		H3("sub"),

		H3("srv"),
	)
	toc.MakeTOC(nav, doc, "h2", "h3", "h4")
	// compose manual page
	NewFile("man.html",
		Html(
			Head(
				Style(
					theme.GoldenSpace(),
					theme.GoishColors(),
					manTheme(),
				),
			),
			Body(doc),
		),
	).SaveTo(".")
}

func manTheme() *web.CSS {
	css := web.NewCSS()
	css.Style("nav ul",
		"list-style-type: none",
	)
	css.Style("li.h3", "margin-left: 2em")
	css.Style("li.h4", "margin-left: 4em")
	return css
}

func must(cmd *exec.Cmd) string {
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Output(2, fmt.Sprint(cmd, err))
		os.Exit(1)
	}
	return string(out)
}
