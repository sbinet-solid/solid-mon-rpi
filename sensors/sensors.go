// Copyright 2017 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sensors exposes sensor data.
package sensors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/go-daq/smbus"
	"github.com/go-daq/smbus/sensor/adc101x"
	"github.com/go-daq/smbus/sensor/at30tse75x"
	"github.com/go-daq/smbus/sensor/bme280"
	"github.com/go-daq/smbus/sensor/hts221"
	"github.com/go-daq/smbus/sensor/tsl2591"
	"gonum.org/v1/plot/plotter"
)

type Sensors struct {
	Timestamp time.Time         `json:"timestamp"`
	Sensors   []Data            `json:"sensors"`
	Labels    map[string][]Type `json:"labels"`
}

type Data struct {
	Name  string  `json:"name"`
	Type  Type    `json:"type"`
	Value float64 `json:"value"`
}

// Type describes the type of data sensor (H,P,T,L,V)
type Type uint8

const (
	InvalidType Type = iota
	Humidity
	Pressure
	Temperature
	Luminosity
	Voltage
)

func (t Type) String() string {
	switch t {
	case InvalidType:
		return "invalid"
	case Humidity:
		return "humidity"
	case Pressure:
		return "pressure"
	case Temperature:
		return "temperature"
	case Luminosity:
		return "luminosity"
	case Voltage:
		return "voltage"
	}
	panic(fmt.Errorf("unknown sensor type %d", t))
}

func (t Type) MarshalJSON() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(t.String())
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// mux maps an I2C channel id to an action register
var mux = [...]uint8{
	0: 0x01,
	1: 0x02,
	2: 0x04,
	3: 0x08,
	4: 0x10,
	5: 0x20,
	6: 0x40,
	7: 0x80,
}

func New(bus *smbus.Conn, addr uint8, descr []Descr) (Sensors, error) {
	data := Sensors{
		Timestamp: time.Now().UTC(),
		Labels:    make(map[string][]Type, len(descr)),
	}
	for _, d := range descr {
		switch d := d.(type) {
		case *DescrADC101x:
			device := ADC101x{}
			if d.Base.I2CAddr == 0 {
				d.Base.I2CAddr = adc101x.DefaultI2CAddr
			}
			err := device.read(bus, d.Base.I2CAddr, addr, mux[d.Base.ChanID], 1024, 3.3)
			if err != nil {
				return data, err
			}
			data.Sensors = append(data.Sensors, Data{
				Name:  d.Descr().Name,
				Type:  Voltage,
				Value: device.Voltage,
			})
			data.Labels[d.Descr().Name] = []Type{Voltage}

		case *DescrAT30TSE:
			device := At30tse75x{}
			if d.I2CAddr == 0 {
				d.I2CAddr = at30tse75x.DefaultI2CAddr
			}
			err := device.read(bus, d.I2CAddr, addr, mux[d.ChanID])
			if err != nil {
				return data, err
			}
			data.Sensors = append(data.Sensors, Data{
				Name:  d.Name,
				Type:  Temperature,
				Value: device.Temp,
			})
			data.Labels[d.Name] = []Type{Temperature}

		case *DescrHTS221:
			device := Hts221{}
			err := device.read(bus, addr, mux[d.ChanID])
			if err != nil {
				return data, err
			}
			data.Sensors = append(data.Sensors, Data{
				Name:  d.Name,
				Type:  Humidity,
				Value: device.Humi,
			})
			data.Sensors = append(data.Sensors, Data{
				Name:  d.Name,
				Type:  Temperature,
				Value: device.Temp,
			})
			data.Labels[d.Name] = []Type{Humidity, Temperature}

		case *DescrBME280:
			device := Bme280{}
			err := device.read(bus, d.I2CAddr, addr, mux[d.ChanID])
			if err != nil {
				return data, err
			}
			data.Sensors = append(data.Sensors, Data{
				Name:  d.Name,
				Type:  Pressure,
				Value: device.Pres,
			})
			data.Labels[d.Name] = append(data.Labels[d.Name], Pressure)

		case *DescrOnBoard:
			{
				device := Bme280{}
				err := device.read(bus, d.I2CAddr, addr, mux[d.ChanID])
				if err != nil {
					return data, err
				}
				data.Sensors = append(data.Sensors, Data{
					Name:  d.Name,
					Type:  Pressure,
					Value: device.Pres,
				})
				data.Labels[d.Name] = append(data.Labels[d.Name], Pressure)
			}
			{
				device := Tsl2591{}
				err := device.read(bus, addr, mux[d.ChanID])
				if err != nil {
					return data, err
				}
				data.Sensors = append(data.Sensors, Data{
					Name:  d.Name,
					Type:  Luminosity,
					Value: device.Lux,
				})
				data.Labels[d.Name] = append(data.Labels[d.Name], Luminosity)
			}
		}
	}
	return data, nil
}

type Tsl2591 struct {
	Lux  float64 `json:"lux"`
	Full uint16  `json:"full"`
	IR   uint16  `json:"ir"`
}

func (tsl *Tsl2591) read(bus *smbus.Conn, addr uint8, ch uint8) error {
	err := bus.WriteReg(addr, 0x04, ch)
	if err != nil {
		log.Printf("tsl-write-reg error: %v", err)
		return err
	}

	dev, err := tsl2591.Open(bus, tsl2591.Addr, tsl2591.IntegTime100ms, tsl2591.GainLow)
	if err != nil {
		log.Printf("tsl-open-bus error: %v", err)
		return err
	}

	full, ir, err := dev.FullLuminosity()
	if err != nil {
		log.Printf("tsl-sample error: %v", err)
		return err
	}

	tsl.Lux = dev.Lux(full, ir)
	tsl.Full = full
	tsl.IR = ir

	return err
}

type Bme280 struct {
	Temp float64 `json:"temp"`
	Hum  float64 `json:"humi"`
	Pres float64 `json:"pres"`
}

func (bme *Bme280) read(bus *smbus.Conn, i2c, addr uint8, ch uint8) error {
	err := bus.WriteReg(addr, 0x04, ch)
	if err != nil {
		log.Printf("write-reg error: %v", err)
		return err
	}

	if i2c == 0 {
		i2c = bme280.I2CAddr
	}
	dev, err := bme280.Open(bus, i2c, bme280.OpSample8)
	if err != nil {
		log.Printf("open-bus error (i2c-addr=0x%x): %v", i2c, err)
		return err
	}

	h, p, t, err := dev.Sample()
	if err != nil {
		log.Printf("sample error: %v", err)
		return err
	}

	const HPa = 1.0 / 100.0
	bme.Hum = h
	bme.Pres = p * HPa
	bme.Temp = t

	return err
}

type ADC101x struct {
	Count   int     `json:"adc"`
	Voltage float64 `json:"voltage"`
}

func (adc *ADC101x) read(bus *smbus.Conn, i2c, daddr uint8, ch uint8, frange int, vdd float64) error {
	err := bus.WriteReg(daddr, 0x04, ch)
	if err != nil {
		log.Printf("adc101x-write-reg error: %v", err)
		return err
	}

	dev, err := adc101x.Open(bus, ch, frange, vdd)
	if err != nil {
		log.Printf("adc101x-open-bus error: %v", err)
		return err
	}

	count, err := dev.ADC()
	if err != nil {
		return err
	}

	voltage, err := dev.Voltage()
	if err != nil {
		return err
	}

	adc.Count = count
	adc.Voltage = voltage
	return nil
}

type At30tse75x struct {
	Temp float64 `json:"temp"`
}

func (at30 *At30tse75x) read(bus *smbus.Conn, i2c, daddr uint8, ch uint8) error {
	err := bus.WriteReg(daddr, 0x04, ch)
	if err != nil {
		log.Printf("at30tse-write-reg error: %v", err)
		return err
	}

	dev, err := at30tse75x.Open(
		bus,
		at30tse75x.I2CAddr(i2c),
		at30tse75x.DevAddr(daddr),
		at30tse75x.EEPROM(4),
	)
	if err != nil {
		log.Printf("at30tse-open-bus error: %v", err)
		return err
	}

	t, err := dev.T()
	if err != nil {
		log.Printf("at30tse-sample error: %v", err)
		return err
	}
	at30.Temp = t
	return nil
}

type Hts221 struct {
	Temp float64 `json:"temp"`
	Humi float64 `json:"humi"`
}

func (hts *Hts221) read(bus *smbus.Conn, addr uint8, ch uint8) error {
	err := bus.WriteReg(addr, 0x04, ch)
	if err != nil {
		log.Printf("hts221-write-reg error: %v", err)
		return err
	}

	dev, err := hts221.Open(bus, hts221.SlaveAddr)
	if err != nil {
		log.Printf("hts221-open-bus error: %v", err)
		return err
	}

	h, t, err := dev.Sample()
	if err != nil {
		log.Printf("hts221-sample error: %v", err)
		return err
	}
	hts.Temp = t
	hts.Humi = h
	return nil
}

type Table []Sensors

func (tbl Table) Data(typ Type, label string) (float64, float64, plotter.XYs) {
	min := +math.MaxFloat64
	max := -math.MaxFloat64
	data := make(plotter.XYs, len(tbl))
	for i, v := range tbl {
		for _, sensor := range v.Sensors {
			if sensor.Type != typ || sensor.Name != label {
				continue
			}
			data[i].X = float64(v.Timestamp.UTC().Unix())
			data[i].Y = sensor.Value
			min = math.Min(min, sensor.Value)
			max = math.Max(max, sensor.Value)
		}
	}
	return min, max, data
}

func (tbl Table) Labels(typ Type) []string {
	var labels []string
	for k, v := range tbl[0].Labels {
		for _, t := range v {
			if typ == t {
				labels = append(labels, k)
			}
		}
	}
	return labels
}
