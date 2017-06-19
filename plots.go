// Copyright 2017 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"image/color"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"go-hep.org/x/hep/hplot"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"github.com/gonum/plot/vg/vgsvg"
	"github.com/sbinet-solid/solid-mon-rpi/sensors"
)

var plotColors = make(map[string]color.Color)

type Plots struct {
	update time.Time
	tile   *hplot.TiledPlot
}

func renderPlot(p *hplot.TiledPlot) string {
	size := 30 * vg.Centimeter
	canvas := vgsvg.New(size, size/vg.Length(math.Phi))
	p.Draw(draw.New(canvas))
	out := new(bytes.Buffer)
	_, err := canvas.WriteTo(out)
	if err != nil {
		panic(err)
	}
	return string(out.Bytes())
}

func newPlots(data []sensors.Sensors) (Plots, error) {
	var (
		ps  Plots
		err error
	)

	ps.update = data[len(data)-1].Timestamp
	const pad = 10
	ps.tile, err = hplot.NewTiledPlot(draw.Tiles{
		Cols:      3,
		Rows:      2,
		PadBottom: pad,
		PadLeft:   pad,
		PadRight:  pad,
		PadTop:    pad,
		PadX:      pad,
		PadY:      pad,
	})
	if err != nil {
		return ps, err
	}

	leg := ps.tile.Plot(0, 2)
	leg.HideAxes()
	labels := make(map[string]int)

	for _, tbl := range []struct {
		pl  *hplot.Plot
		typ sensors.Type
	}{
		{ps.tile.Plot(0, 0), sensors.Humidity},
		{ps.tile.Plot(0, 1), sensors.Pressure},
		{ps.tile.Plot(1, 0), sensors.Temperature},
		{ps.tile.Plot(1, 1), sensors.Luminosity},
	} {
		tbl.pl.Title.Text = strings.Title(tbl.typ.String())
		tbl.pl.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}
		tbl.pl.X.Tick.Label.Rotation = math.Pi / 4
		tbl.pl.X.Tick.Label.YAlign = draw.YTop
		tbl.pl.X.Tick.Label.XAlign = draw.XRight

		err = setupPlot(tbl.pl, leg, &labels, data, tbl.typ)
		if err != nil {
			return ps, err
		}
	}

	ps.tile.Plots[5] = nil

	return ps, err
}

func (ps *Plots) MarshalJSON() ([]byte, error) {
	var raw struct {
		Plot   string `json:"plot"`
		Update string `json:"update"`
	}

	raw.Plot = renderPlot(ps.tile)
	raw.Update = ps.update.Format("2006-01-02 15:04:05 (MST)")

	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(raw)
	if err != nil {
		log.Printf("plots-marshal: %v", err)
		return nil, err
	}
	return buf.Bytes(), nil
}

func setupPlot(pl, leg *hplot.Plot, names *map[string]int, table sensors.Table, typ sensors.Type) error {
	min := +math.MaxFloat64
	max := -math.MaxFloat64
	{
		labels := table.Labels(typ)
		sort.Strings(labels)
		for k := range labels {
			label := labels[k]
			ymin, ymax, data := table.Data(typ, label)
			min = math.Min(min, ymin)
			max = math.Max(max, ymax)
			lines, err := plotter.NewLine(data)
			if err != nil {
				return err
			}
			lines.Color = plotColors[label]
			pl.Add(lines)
			if _, dup := (*names)[label]; !dup {
				leg.Legend.Add(label, lines)
				(*names)[label] = 1
			}
		}
	}

	if typ == sensors.Pressure {
		// FIXME(sbinet): hack to work around https://github.com/gonum/plot/issues/366
		pl.Y.Min = min - 0.5
		pl.Y.Max = max + 0.5
	}

	pl.Add(plotter.NewGrid())

	return nil
}
