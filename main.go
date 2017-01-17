// Copyright 2017 The tcp-srv Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"log"
	"net"
	"time"

	"github.com/go-daq/smbus"
	"github.com/sbinet-solid/tcp-srv/sensors"
)

var (
	addr    = flag.String("addr", ":10000", "[ip]:port for TCP server")
	busID   = flag.Int("bus-id", 0x1, "SMBus ID number (/dev/i2c-[ID]")
	busAddr = flag.Int("bus-addr", 0x70, "SMBus address to read/write")
	freq    = flag.Duration("freq", 2*time.Second, "data polling interval")

	datac = make(chan sensors.Sensors)
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

	go daq()

	for {
		conn, err := srv.Accept()
		if err != nil {
			log.Printf("error accepting connection: %v\n", err)
			continue
		}
		go handle(conn)
	}
}

func daq() {
	bus, err := smbus.Open(*busID, uint8(*busAddr))
	if err != nil {
		log.Fatalf("error opening smbus(id=%v addr=%v): %v\n", *busID, *busAddr, err)
		return
	}
	defer bus.Close()

	tick := time.NewTicker(*freq)
	defer tick.Stop()

	for range tick.C {
		data, err := fetchData(bus)
		if err != nil {
			log.Printf("error fetching data: $v\n", err)
			continue
		}

		log.Printf("daq: %+v\n", data)
		select {
		case datac <- data:
		default:
			// nobody is listening
			// drop it on the floor
		}
	}
}

func handle(conn net.Conn) {
	defer conn.Close()

	log.Printf("connection from: %v\n", conn.RemoteAddr())

	for data := range datac {
		var hdr [4]byte
		var buf bytes.Buffer
		err := json.NewEncoder(&buf).Encode(data)
		if err != nil {
			log.Printf("error sending json data: %v\n", err)
			return
		}
		binary.LittleEndian.PutUint32(hdr[:], uint32(buf.Len()))
		_, err = conn.Write(hdr[:])
		if err != nil {
			log.Printf("error sending header: %v\n", err)
			return
		}

		_, err = buf.WriteTo(conn)
		if err != nil {
			log.Printf("error sending data: %v\n", err)
			return
		}

	}
}

func fetchData(bus *smbus.Conn) (sensors.Sensors, error) {
	data, err := sensors.New(bus, uint8(*busAddr))
	if err != nil {
		return data, err
	}

	return data, nil
}
