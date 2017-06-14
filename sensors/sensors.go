// Copyright 2017 The tcp-srv Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sensors exposes sensor data.
package sensors

import (
	"fmt"
	"log"
	"time"

	"github.com/go-daq/smbus"
	"github.com/go-daq/smbus/sensor/at30tse75x"
	"github.com/go-daq/smbus/sensor/bme280"
	"github.com/go-daq/smbus/sensor/tsl2591"
)

type Sensors struct {
	Timestamp time.Time `json:"timestamp"`
	Tsl       Tsl       `json:"tsl"`
	Bme       Bme       `json:"bme280"`
	At30tse   At30tse   `json:"at30tse"`
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
	err = s.Tsl.read(bus, addr, 0x80)
	if err != nil {
		return fmt.Errorf("tsl error: %v", err)
	}
	err = s.Bme.read(bus, addr, 0x80)
	if err != nil {
		return fmt.Errorf("bme error: %v", err)
	}
	err = s.At30tse.read(bus, addr, 0x08)
	if err != nil {
		return fmt.Errorf("at30tse error: %v", err)
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

type Bme struct {
	Temp float64 `json:"temp"`
	Hum  float64 `json:"hum"`
	Pres float64 `json:"pres"`
}

func (bme *Bme) read(bus *smbus.Conn, addr uint8, ch uint8) error {
	err := bus.WriteReg(addr, 0x04, ch)
	if err != nil {
		log.Printf("write-reg error: %v", err)
		return err
	}

	dev, err := bme280.Open(bus, bme280.I2CAddr, bme280.OpSample8)
	if err != nil {
		log.Printf("open-bus error (i2c-addr=0x%x): %v", bme280.I2CAddr, err)
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

type At30tse struct {
	Temp float64 `json:"temp"`
}

func (at30 *At30tse) read(bus *smbus.Conn, addr uint8, ch uint8) error {
	err := bus.WriteReg(addr, 0x04, ch)
	if err != nil {
		log.Printf("at30tse-write-reg error: %v", err)
		return err
	}

	const eeprom = 4
	dev, err := at30tse75x.Open(bus, 0, eeprom)
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
