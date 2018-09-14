// Copyright 2018 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Command solid-mgr-rpi updates the RPi read-only configuration with a new overlay.
package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	bootCmdLine = "/tmp/boot/cmdline.txt" // FIXME(sbinet)
)

func main() {
	log.SetPrefix("rpi-mgr: ")
	log.SetFlags(0)

	flag.Parse()

	host := flag.Arg(0)
	cmd := flag.Arg(1)
	log.Printf("host=%q", host)
	log.Printf("cmd=%q", cmd)

	switch cmd {
	case "ro":
		err := cmdRO(host)
		if err != nil {
			log.Fatal(err)
		}
	case "rw":
		err := cmdRW(host)
		if err != nil {
			log.Fatal(err)
		}
	default:
		log.Fatalf("unknown command %q", cmd)
	}
}

func removeOverlay(p []byte) ([]byte, error) {
	o := bytes.Replace(p, []byte("boot=overlay "), []byte(""), -1)
	o = bytes.Replace(o, []byte(" boot=overlay"), []byte(""), -1)
	o = bytes.Replace(o, []byte("  "), []byte(" "), -1)
	return o, nil
}

func addOverlay(p []byte) ([]byte, error) {
	o, err := removeOverlay(p)
	if err != nil {
		return nil, err
	}
	o = append([]byte("boot=overlay "), o...)
	return o, nil
}

func cmdRO(host string) error {
	buf, err := fetchContent(host)
	if err != nil {
		return err
	}

	ro, err := addOverlay(buf)
	if err != nil {
		return errors.Wrapf(err, "could not add boot=overlay to %s:%s", host, bootCmdLine)
	}

	err = putContent(host, ro)
	if err != nil {
		return err
	}

	return nil
}

func cmdRW(host string) error {
	buf, err := fetchContent(host)
	if err != nil {
		return err
	}

	ro, err := removeOverlay(buf)
	if err != nil {
		return errors.Wrapf(err, "could not remove boot=overlay from %s:%s", host, bootCmdLine)
	}

	err = putContent(host, ro)
	if err != nil {
		return err
	}

	return nil
}

func fetchContent(host string) ([]byte, error) {
	buf := new(bytes.Buffer)
	cmd := exec.Command("ssh", host, "--", "cat", bootCmdLine)
	cmd.Stdout = buf
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return nil, errors.Wrapf(err, "could not extract %q from %q", bootCmdLine, host)
	}

	log.Printf("content: %q", buf.String())

	return buf.Bytes(), nil
}

func putContent(host string, data []byte) error {
	tmp, err := ioutil.TempDir("", "solid-mgr-boot-")
	if err != nil {
		return errors.Wrapf(err, "could not create temporary directory")
	}
	defer os.RemoveAll(tmp)

	fname := filepath.Join(tmp, "cmdline.txt")
	err = ioutil.WriteFile(fname, data, 0644)
	if err != nil {
		return errors.Wrapf(err, "could not create new ro cmdline.txt file")
	}

	cmd := exec.Command("scp", fname, host+":"+bootCmdLine)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
