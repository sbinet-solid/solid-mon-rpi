// Copyright 2018 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"flag"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	log.SetPrefix("release: ")
	log.SetFlags(0)

	flag.Parse()

	tag := flag.Arg(0)

	os.Setenv("GO111MODULE", "on")

	run("go", "generate")
	run("go", "get", "-v", "./...")

	oname := filepath.Join(".", "releases", tag, "solid-mon-rpi-linux-arm.exe")
	err := os.MkdirAll(filepath.Dir(oname), 0755)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("create an ARM-based executable for RPi3...")
	cmd := exec.Command("go", "build", "-v", "-o", oname)
	cmd.Env = append([]string{
		"GOARCH=arm",
		"GOARM=7",
		"GO111MODULE=on",
	}, os.Environ()...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	err = os.Chmod(oname, 0755)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("create an ARM-based executable for RPi3... [done]")

	dir := filepath.Dir(oname)
	log.Printf("xfer %q to CERN...", dir)
	run("scp", "-r", dir, "lxplus.cern.ch:www/solid-mon-rpi/.")
}

func run(cmd string, args ...string) {
	c := exec.Command(cmd, args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	err := c.Run()
	if err != nil {
		log.Fatal(err)
	}
}
