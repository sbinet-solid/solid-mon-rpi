// Copyright 2017 The tcp-srv Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"flag"
	"log"
	"math/rand"
	"net"
	"time"

	"github.com/go-daq/smbus"
	"github.com/go-daq/smbus/sensor/sht3x"
	"github.com/go-daq/smbus/sensor/tsl2591"
)

var (
	addr    = flag.String("addr", ":10000", "[ip]:port for TCP server")
	busID   = flag.Int("bus-id", 0x1, "SMBus ID number (/dev/i2c-[ID]")
	busAddr = flag.Int("bus-addr", 0x70, "SMBus address to read/write")
	debug   = flag.Bool("dbg", false, "(debugging only)")
)

func main() {
	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("tcp-srv ")

	log.Printf("starting up server on: %v\n", *addr)

	srv, err := net.Listen("tcp", *addr)
	if err != nil {
		log.Fatal(err)
	}

	if *debug {
		go runClient(*addr)
	}

	for {
		conn, err := srv.Accept()
		if err != nil {
			log.Printf("error accepting connection: %v\n", err)
			continue
		}
		go handle(conn)
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	log.Printf("connection from: %v\n", conn.RemoteAddr())

	data, err := fetchData()
	if err != nil {
		log.Printf("error fetching data: %v\n", err)
		return
	}

	err = json.NewEncoder(conn).Encode(data)
	if err != nil {
		log.Printf("error sending json data: %v\n", err)
		return
	}
}

func fetchData() (Sensors, error) {
	if *debug {
		return genData()
	}
	bus, err := smbus.Open(*busID, uint8(*busAddr))
	if err != nil {
		return Sensors{}, err
	}
	defer bus.Close()

	data := Sensors{Timestamp: time.Now().UTC()}
	err = data.read(bus, uint8(*busAddr))
	if err != nil {
		return data, err
	}

	return data, nil
}

type Sensors struct {
	Timestamp time.Time `json:"timestamp"`
	Tsl       Tsl       `json:"tsl"`
	Sht31     Sht31     `json:"sht31"`
	Si7021    [2]Si7021 `json:"si7021"`
	Bme       Bme       `json:"bme280"`
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

	return err
}

func runClient(addr string) {
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()

	for range tick.C {

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Fatalf("client error: %v\n", err)
		}
		var data Sensors
		err = json.NewDecoder(conn).Decode(&data)
		conn.Close()
		if err != nil {
			log.Fatalf("client decoding error: %v\n", err)
		}
		log.Printf("--> recv: %v\n", data)
	}
}

func genData() (Sensors, error) {
	var err error
	data := Sensors{
		Timestamp: time.Now().UTC(),
		Tsl: Tsl{
			Lux:  rand.Float64(),
			Full: uint16(rand.Uint32()),
			IR:   uint16(rand.Uint32()),
		},
		Sht31: Sht31{
			Temp: rand.Float64(),
			Hum:  rand.Float64(),
		},
		Si7021: [2]Si7021{
			{
				Temp: rand.Float64(),
				Hum:  rand.Float64(),
			},
			{
				Temp: rand.Float64(),
				Hum:  rand.Float64(),
			},
		},
		Bme: Bme{
			Temp: rand.Float64(),
			Hum:  rand.Float64(),
			Pres: rand.Float64(),
		},
	}
	return data, err
}
