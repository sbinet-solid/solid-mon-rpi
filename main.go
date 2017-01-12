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
)

var (
	addr  = flag.String("addr", ":10000", "[ip]:port for TCP server")
	debug = flag.Bool("dbg", false, "(debugging only)")
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
	}
}

func fetchData() (Sensors, error) {
	if *debug {
		return genData()
	}
	panic("not implemented")
}

type Sensors struct {
	Timestamp time.Time `json:"timestamp"`
	Tsl       Tsl       `json:"tsl"`
	Sht31     Sht31     `json:"sht31"`
	Si7021    [2]Si7021 `json:"si7021"`
	Bme       Bme       `json:"bme280"`
}

type Tsl struct {
	Lux  float64 `json:"lux"`
	Full float64 `json:"full"`
	IR   float64 `json:"ir"`
}

type Sht31 struct {
	Temp float64 `json:"temp"`
	Hum  float64 `json:"hum"`
}

type Si7021 struct {
	Temp float64 `json:"temp"`
	Hum  float64 `json:"hum"`
}

type Bme struct {
	Temp float64 `json:"temp"`
	Hum  float64 `json:"hum"`
	Pres float64 `json:"pres"`
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
			Full: rand.Float64(),
			IR:   rand.Float64(),
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
