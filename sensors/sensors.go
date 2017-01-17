// Copyright 2017 The tcp-srv Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sensors exposes sensor data.
package sensors

import (
	"time"

	"github.com/go-daq/smbus"
	"github.com/go-daq/smbus/sensor/bme280"
	"github.com/go-daq/smbus/sensor/sht3x"
	"github.com/go-daq/smbus/sensor/si7021"
	"github.com/go-daq/smbus/sensor/tsl2591"
)

type Sensors struct {
	Timestamp time.Time `json:"timestamp"`
	Tsl       Tsl       `json:"tsl"`
	Sht31     Sht31     `json:"sht31"`
	Si7021    [2]Si7021 `json:"si7021"`
	Bme       Bme       `json:"bme280"`
}

func New(bus *smbus.Conn, addr uint8) (Sensors, error) {
	data := Sensors{
		Timestamp: time.Now().UTC(),
	}
	err := data.read(bus, addr)
	if err != nil {
		return Sensors{}, err
	}
	return data, nil
}

func (s *Sensors) read(bus *smbus.Conn, addr uint8) error {
	var err error
	err = s.Tsl.read(bus, addr, 0x01)
	if err != nil {
		return err
	}
	err = s.Sht31.read(bus, addr, 0x02)
	if err != nil {
		return err
	}
	err = s.Si7021[0].read(bus, addr, 0x04)
	if err != nil {
		return err
	}
	err = s.Si7021[1].read(bus, addr, 0x08)
	if err != nil {
		return err
	}
	err = s.Bme.read(bus, addr, 0x10)
	if err != nil {
		return err
	}
	return err
}

type Tsl struct {
	Lux  float64 `json:"lux"`
	Full uint16  `json:"full"`
	IR   uint16  `json:"ir"`
}

func (tsl *Tsl) read(bus *smbus.Conn, addr uint8, ch uint8) error {
	err := bus.WriteReg(addr, 0x04, ch)
	if err != nil {
		return err
	}

	dev, err := tsl2591.Open(bus, tsl2591.Addr, tsl2591.IntegTime100ms, tsl2591.GainLow)
	if err != nil {
		return err
	}

	full, ir, err := dev.FullLuminosity()
	if err != nil {
		return err
	}

	tsl.Lux = dev.Lux(full, ir)
	tsl.Full = full
	tsl.IR = ir

	return err
}

type Sht31 struct {
	Temp float64 `json:"temp"`
	Hum  float64 `json:"hum"`
}

func (sht *Sht31) read(bus *smbus.Conn, addr uint8, ch uint8) error {
	err := bus.WriteReg(addr, 0x04, ch)
	if err != nil {
		return err
	}

	dev, err := sht3x.Open(bus, sht3x.I2CAddr)
	if err != nil {
		return err
	}

	t, rh, err := dev.Sample()
	if err != nil {
		return err
	}

	sht.Temp = t
	sht.Hum = rh

	err = dev.ClearStatus()
	if err != nil {
		return err
	}

	return nil
}

type Si7021 struct {
	Temp float64 `json:"temp"`
	Hum  float64 `json:"hum"`
}

func (si *Si7021) read(bus *smbus.Conn, addr uint8, ch uint8) error {
	err := bus.WriteReg(addr, 0x04, ch)
	if err != nil {
		return err
	}

	dev, err := si7021.Open(bus, 0x40)
	if err != nil {
		return err
	}

	h, err := dev.Humidity()
	if err != nil {
		return err
	}

	t, err := dev.Temperature()
	if err != nil {
		return err
	}

	si.Temp = t
	si.Hum = h
	return err
}

type Bme struct {
	Temp float64 `json:"temp"`
	Hum  float64 `json:"hum"`
	Pres float64 `json:"pres"`
}

func (bme *Bme) read(bus *smbus.Conn, addr uint8, ch uint8) error {
	err := bus.WriteReg(addr, 0x04, ch)
	if err != nil {
		return err
	}

	dev, err := bme280.Open(bus, bme280.I2CAddr, bme280.OpSample8)
	if err != nil {
		return err
	}

	h, p, t, err := dev.Sample()
	if err != nil {
		return err
	}

	const HPa = 1.0 / 100.0
	bme.Hum = h
	bme.Pres = p * HPa
	bme.Temp = t

	return err
}
