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
	"time"

	"go-hep.org/x/hep/hplot"

	"github.com/gonum/plot"
	"github.com/gonum/plot/plotter"
	"github.com/gonum/plot/vg"
	"github.com/gonum/plot/vg/draw"
	"github.com/gonum/plot/vg/vgsvg"
	"github.com/sbinet-solid/solid-mon-rpi/sensors"
)

var (
	plotColors = map[string]color.Color{
		"BME280":  color.NRGBA{255, 0, 0, 128},
		"AT30TSE": color.NRGBA{0, 255, 0, 128},
		"HTS221":  color.NRGBA{0, 0, 255, 128},
		"TSL2591": color.Black,
	}
)

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
		name  string
		pl    *hplot.Plot
		setup func(p, leg *hplot.Plot, names map[string]int, table []sensors.Sensors) error
	}{
		{"Humidity", ps.tile.Plot(0, 0), setupPlotHumidity},
		{"Pressure", ps.tile.Plot(0, 1), setupPlotPressure},
		{"Temperature", ps.tile.Plot(1, 0), setupPlotTemp},
		{"Lux", ps.tile.Plot(1, 1), setupPlotLux},
	} {
		tbl.pl.Title.Text = tbl.name
		tbl.pl.X.Tick.Marker = plot.TimeTicks{Format: "2006-01-02\n15:04:05"}
		tbl.pl.X.Tick.Label.Rotation = math.Pi / 4
		tbl.pl.X.Tick.Label.YAlign = draw.YTop
		tbl.pl.X.Tick.Label.XAlign = draw.XRight

		err = tbl.setup(tbl.pl, leg, labels, data)
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

func setupPlotHumidity(pl, leg *hplot.Plot, names map[string]int, table []sensors.Sensors) error {
	{
		data := make(plotter.XYs, len(table))
		for i, v := range table {
			data[i].X = float64(v.Timestamp.UTC().Unix())
			data[i].Y = v.Bme280.Hum
		}
		lines, err := plotter.NewLine(data)
		if err != nil {
			return err
		}
		name := "BME280"
		lines.Color = plotColors[name]
		pl.Add(lines)
		if _, dup := names[name]; !dup {
			leg.Legend.Add(name, lines)
			names[name] = 1
		}
	}

	{
		data := make(plotter.XYs, len(table))
		for i, v := range table {
			data[i].X = float64(v.Timestamp.UTC().Unix())
			data[i].Y = v.Hts221.Humi
		}
		lines, err := plotter.NewLine(data)
		if err != nil {
			return err
		}
		name := "HTS221"
		lines.Color = plotColors[name]
		pl.Add(lines)
		if _, dup := names[name]; !dup {
			leg.Legend.Add(name, lines)
			names[name] = 1
		}
	}

	pl.Add(plotter.NewGrid())

	return nil
}

func setupPlotPressure(pl, leg *hplot.Plot, names map[string]int, table []sensors.Sensors) error {
	min := +math.MaxFloat64
	max := -math.MaxFloat64
	{
		data := make(plotter.XYs, len(table))
		for i, v := range table {
			data[i].X = float64(v.Timestamp.UTC().Unix())
			data[i].Y = v.Bme280.Pres
			p := v.Bme280.Pres
			max = math.Max(max, p)
			min = math.Min(min, p)
		}
		lines, err := plotter.NewLine(data)
		if err != nil {
			return err
		}
		name := "BME280"
		lines.Color = plotColors[name]
		pl.Add(lines)
		if _, dup := names[name]; !dup {
			leg.Legend.Add(name, lines)
			names[name] = 1
		}
	}

	// FIXME(sbinet): hack to work around https://github.com/gonum/plot/issues/366
	pl.Y.Min = min - 0.5
	pl.Y.Max = max + 0.5

	pl.Add(plotter.NewGrid())

	return nil
}

func setupPlotTemp(pl, leg *hplot.Plot, names map[string]int, table []sensors.Sensors) error {
	{
		data := make(plotter.XYs, len(table))
		for i, v := range table {
			data[i].X = float64(v.Timestamp.UTC().Unix())
			data[i].Y = v.Bme280.Temp
		}
		lines, err := plotter.NewLine(data)
		if err != nil {
			return err
		}
		name := "BME280"
		lines.Color = plotColors[name]
		pl.Add(lines)
		if _, dup := names[name]; !dup {
			leg.Legend.Add(name, lines)
			names[name] = 1
		}
	}

	{
		data := make(plotter.XYs, len(table))
		for i, v := range table {
			data[i].X = float64(v.Timestamp.UTC().Unix())
			data[i].Y = v.Hts221.Temp
		}
		lines, err := plotter.NewLine(data)
		if err != nil {
			return err
		}
		name := "HTS221"
		lines.Color = plotColors[name]
		pl.Add(lines)
		if _, dup := names[name]; !dup {
			leg.Legend.Add(name, lines)
			names[name] = 1
		}
	}

	{
		data := make(plotter.XYs, len(table))
		for i, v := range table {
			data[i].X = float64(v.Timestamp.UTC().Unix())
			data[i].Y = v.At30tse75x.Temp
		}
		lines, err := plotter.NewLine(data)
		if err != nil {
			return err
		}
		name := "AT30TSE"
		lines.Color = plotColors[name]
		pl.Add(lines)
		if _, dup := names[name]; !dup {
			leg.Legend.Add(name, lines)
			names[name] = 1
		}
	}

	pl.Add(plotter.NewGrid())

	return nil
}

func setupPlotLux(pl, leg *hplot.Plot, names map[string]int, table []sensors.Sensors) error {
	{
		data := make(plotter.XYs, len(table))
		for i, v := range table {
			data[i].X = float64(v.Timestamp.UTC().Unix())
			data[i].Y = v.Tsl2591.Lux
		}
		lines, err := plotter.NewLine(data)
		if err != nil {
			return err
		}
		name := "TSL2591"
		lines.Color = plotColors[name]
		pl.Add(lines)
		if _, dup := names[name]; !dup {
			leg.Legend.Add(name, lines)
			names[name] = 1
		}
	}

	pl.Add(plotter.NewGrid())
	return nil
}
