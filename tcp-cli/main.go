// Copyright 2017 The tcp-srv Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"net"
	"time"
)

var (
	addr  = flag.String("addr", "clrmedaq02.in2p3.fr:10000", "[ip]:port for TCP server")
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
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()

	for range tick.C {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			log.Fatalf("client error: %v\n", err)
		}
		var buf [1024]byte
		n, err := conn.Read(buf[:])
		conn.Close()

		if err != nil {
			log.Fatalf("client error: %v\n", err)
		}
		log.Printf("read %d bytes: %v\n", n, string(buf[:]))
	}
}
