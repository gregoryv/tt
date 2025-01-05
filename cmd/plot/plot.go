package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"os"

	"github.com/gregoryv/cmdline"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/plotutil"
	"gonum.org/v1/plot/vg"
)

func main() {
	var (
		cli      = cmdline.NewBasicParser()
		filename = cli.NamedArg("FILENAME").String("")
	)
	cli.Parse()

	log.SetFlags(0)

	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	p := plot.New()
	p.Title.Text = fmt.Sprintf("tt mon; %s", title(data))
	p.X.Label.Text = "Seconds"

	{
		l := &plotter.Line{
			XYs:       points(data, "MemAlloc"),
			LineStyle: plotter.DefaultLineStyle,
		}
		l.Color = plotutil.Color(0)
		p.Add(l)
		p.Legend.Add("Allocated memory (KB)", l)

	}
	{
		l := &plotter.Line{
			XYs:       points(data, "ConnActive"),
			LineStyle: plotter.DefaultLineStyle,
		}
		l.Color = plotutil.Color(2)
		p.Add(l)
		p.Legend.Add("Connections", l)
	}
	// Save the plot to a PNG file.
	if err := p.Save(8*vg.Inch, 4*vg.Inch, "/tmp/ttplot.png"); err != nil {
		panic(err)
	}
}

func title(data []byte) string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var (
		date, time, rev                               string
		connCount, connActive, numGoroutine, memAlloc int
	)
	format := "%s %s %s %v %v %v %v"
	scanner.Scan() // skip first line
	scanner.Scan()
	line := scanner.Text()
	fmt.Sscanf(line, format,
		&date,
		&time,
		&rev,
		&connCount,
		&connActive,
		&memAlloc,
		&numGoroutine,
	)
	return fmt.Sprintf("%s %s %s", date, time, rev)
}

func points(data []byte, key string) plotter.XYs {
	pts := make(plotter.XYs, 0)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var (
		date, time, rev                               string
		connCount, connActive, numGoroutine, memAlloc int
	)
	format := "%s %s %s %v %v %v %v"
	scanner.Scan() // skip first line
	var i int
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Sscanf(line, format,
			&date,
			&time,
			&rev,
			&connCount,
			&connActive,
			&memAlloc,
			&numGoroutine,
		)
		p := plotter.XY{
			X: float64(i),
		}
		switch key {
		case "ConnCount":
			p.Y = float64(connCount)
		case "ConnActive":
			p.Y = float64(connActive)
		case "MemAlloc":
			p.Y = float64(memAlloc / 1024)
		default:
			panic("points(" + key + ")")
		}
		pts = append(pts, p)
		i++
	}
	return pts
}

// randomPoints returns some random x, y points.
func randomPoints(n int) plotter.XYs {
	pts := make(plotter.XYs, n)
	for i := range pts {
		if i == 0 {
			pts[i].X = rand.Float64()
		} else {
			pts[i].X = pts[i-1].X + rand.Float64()
		}
		pts[i].Y = pts[i].X + 10*rand.Float64()
	}
	return pts
}
