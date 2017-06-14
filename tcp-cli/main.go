// Copyright 2017 The tcp-srv Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/sbinet-solid/tcp-srv/sensors"
)

var (
	addr  = flag.String("addr", "clrmedaq01.in2p3.fr:8080", "[ip]:port for TCP server")
	debug = flag.Bool("dbg", false, "(debugging only)")
)

func main() {
	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("tcp-cli ")

	log.Printf("dialing: %v\n", *addr)

	runClient(*addr)
}

func runClient(addr string) {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		log.Fatalf("client error: %v\n", err)
	}
	defer conn.Close()

	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()

	r := conn
	for range tick.C {
		var hdr uint32
		err = binary.Read(r, binary.LittleEndian, &hdr)
		if err != nil {
			log.Fatalf("error reading header: %v\n", err)
		}

		buf := make([]byte, int(hdr))
		_, err = io.ReadFull(r, buf)
		if err != nil {
			log.Fatalf("error reading data: %v\n", err)
		}

		var data sensors.Sensors
		err = json.NewDecoder(bytes.NewReader(buf)).Decode(&data)
		if err != nil {
			log.Fatalf("json dec-error: %v\n", err)
		}
		err = json.NewEncoder(os.Stdout).Encode(data)
		if err != nil {
			log.Fatalf("json enc-error: %v\n", err)
		}
	}
}
