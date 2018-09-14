// Copyright 2018 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"testing"
)

func TestBootOverlay(t *testing.T) {
	const (
		orig = `boot=overlay dwc_otg.lpm_enable=0 console=serial0,115200 console=tty1 root=/dev/mmcblk0p7 rootfstype=ext4 elevator=deadline fsck.repair=yes rootwait`
		want = `dwc_otg.lpm_enable=0 console=serial0,115200 console=tty1 root=/dev/mmcblk0p7 rootfstype=ext4 elevator=deadline fsck.repair=yes rootwait`
	)

	boot := []byte(orig)
	modified, err := removeOverlay(boot)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(modified, []byte(want)) {
		t.Fatalf("got: %q\nwant:%q\n", string(modified), string(want))
	}

	ronly, err := addOverlay(modified)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(ronly, boot) {
		t.Fatalf("got: %q\nwant:%q\n", string(ronly), string(boot))
	}
}
