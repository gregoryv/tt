// Command feature documents tt features
//
// The command executes each feature and writes the documentation for
// it to stdout.
package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
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

	// compose manual page
	NewFile("man.html",
		Html(
			Head(
				Title("tt - manual"),
				Style(
					theme.GoldenSpace(),
					theme.GoishColors(),
					manTheme(),
				),
			),
			Body(Manual()),
		),
	).SaveTo(".")
}

func Manual() *Element {
	nav := Nav(A(Name("toc")), A(Href("#toc"), B("Table of contents")))
	doc := Wrap(
		Header(
			time.Now().Format("2006-01-02 15:04:05"),
		),

		Article(
			H1(A(Href("man.html"), "Telemetry Transfer (tt) - manual")),

			P(`The tt command provides a mqtt-v5 server and pub/sub
			clients for general use. Project source is found at
			gregoryv/tt. The intention of this manual is two fold,
			describe the features of each command and verify that the
			features conform to the specification where possible.`),

			nav,

			H2("Options"),

			P(`The options section describes shared options.`),

			Pre(Class("cmd"),
				must(exec.Command("tt", "--help")),
			),

			H2("Commands"),

			// ----------------------------------------
			// pub command
			H3("pub"),

			P(`Client for publishing application messages to a mqtt-v5 server.`),

			H4("Publish QoS 0"),

			P(`Default command publishes a predefined message to a
			local server.`),

			Div(Class("figure"),
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
			),

			Pre(Class("cmd"), must(exec.Command("tt"))),

			// ----------------------------------------
			// sub command
			H3("sub"),

			P(`Client subscribing to topic patterns.`),

			H4("Subscribe all topics"),

			P(`Default sub command subscribes to all topics and blocks
			until interrupted.`),

			Pre(Class("cmd"), must(exec.Command("tt", "sub"))),

			// ----------------------------------------
			// srv command
			H3("srv"),

			H4("Serve clients on tcp://"),

			Pre(Class("cmd"), must(exec.Command("tt", "srv", "-b", "tcp://localhost:9983"))),

			H2("Full features"),
			H3("pub-sub sequence"),

			P(`Clients connect to the same server. First client
            subscribes to topic filter gopher/+, and second publishes
            a message to topic name gopher/pink. Log messages are
            grouped for clarity. See `,
				A(
					Href("https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Message_delivery_retry"),
					"4.4 Message delivery retry wip ",
				),
			),

			Pre(Class("cmd"), func() interface{} {
				var buf, a, b bytes.Buffer
				server := "tcp://localhost:9983"
				{ // start server
					cmd := exec.Command("tt", "srv", "-b", server)
					cmd.Stdout = &buf
					cmd.Stderr = &buf
					tidyGobin(&buf, cmd, "&")
					if err := cmd.Start(); err != nil {
						log.Println(cmd, err)
					}
					defer cmd.Process.Signal(os.Interrupt)
				}
				{ // start sub client
					<-time.After(3 * time.Millisecond)
					cmd := exec.Command("tt", "sub", "-t", "gopher/+", "-s", server)
					cmd.Stdout = &a
					cmd.Stderr = &a
					tidyGobin(&a, cmd, "&")
					if err := cmd.Start(); err != nil {
						log.Println(cmd, err)
					}
					defer cmd.Process.Signal(os.Interrupt)
				}
				{ // publish message with client B
					<-time.After(3 * time.Millisecond)
					cmd := exec.Command("tt", "pub", "-s", server)
					cmd.Stdout = &b
					cmd.Stderr = &b
					tidyGobin(&b, cmd, "")
					if err := cmd.Run(); err != nil {
						log.Println(cmd, err)
					}
				}

				return strings.Join([]string{
					buf.String(),
					a.String(),
					b.String(),
				}, "\n")
			}()),
			//
		),
	)
	// must link these first as the words are part in secondary links
	LinkAll(doc, map[string]string{
		"mqtt-v5": "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html",
	})
	links := map[string]string{
		"gregoryv/tt": "https://github.com/gregoryv/tt",
		"CONNECT":     "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901033",
		"CONNACK":     "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901074",
		"SUBSCRIBE":   "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901161",
		"SUBACK":      "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901171",
		"PUBLISH":     "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901100",
		"PUBACK":      "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901121",
		"PUBREC":      "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901131",
		"PUBREL":      "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901141",
		"UNSUBSCRIBE": "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901179",
		"UNSUBACK":    "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901187",
		"PINGREQ":     "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901195",
		"PINGRESP":    "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901200",
		"DISCONNECT":  "https://docs.oasis-open.org/mqtt/mqtt/v5.0/os/mqtt-v5.0-os.html#_Toc3901205",
	}
	LinkAll(doc, links)

	toc.MakeTOC(nav, doc, "h2", "h3", "h4")
	return doc
}

// tidyGobin removes the first occurence of GOBIN path in cmd and
// writes the result to the given buffer. The suffix can be used to
// e.g. add a "&" to indicate it's been executed in the background.
func tidyGobin(buf *bytes.Buffer, cmd *exec.Cmd, suffix string) {
	c := strings.Replace(cmd.String(), os.Getenv("GOBIN")+"/", "", 1)
	buf.WriteString(fmt.Sprintf("$ %s%s\n", c, suffix))
}

func manTheme() *web.CSS {
	css := web.NewCSS()
	css.Style("body",
		"max-width: 19cm",
		"margin: auto auto",
		"font-family: sans-serif",
	)
	css.Style("h1,h2,h3,h4,h5",
		"font-family: serif",
	)
	css.Style("header",
		"text-align: right",
	)

	css.Style("nav ul",
		"list-style-type: none",
	)
	css.Style("div.figure",
		"width: 100%",
		"text-align: center",
	)
	css.Style("li.h3", "margin-left: 2em")
	css.Style("li.h4", "margin-left: 4em")
	css.Style("pre.cmd",
		"border-left: 7px #727272 solid",
		"padding: .6em 1.6em .6em 1.6em",
		"background-color: #eaeaea",
	)
	css.Style(".fail", "color: red")
	return css
}

func must(cmd *exec.Cmd) interface{} {
	var buf bytes.Buffer
	c := strings.Replace(cmd.String(), os.Getenv("GOBIN")+"/", "", 1)
	buf.WriteString(fmt.Sprintf("$ %s\n", c))

	cmd.Stdout = &buf
	cmd.Stderr = &buf

	if err := cmd.Start(); err != nil {
		log.Output(2, fmt.Sprint(cmd, err))
	}
	go time.AfterFunc(time.Second, func() { cmd.Process.Signal(os.Interrupt) })
	state, err := cmd.Process.Wait()
	if !state.Success() || err != nil {
		return Span(Class("fail"), buf.String(), state.String())
	}

	return buf.String()
}
