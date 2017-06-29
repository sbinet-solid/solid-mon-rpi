// Copyright 2017 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sensors

import (
	"bytes"
	"encoding/xml"
	"reflect"
	"testing"
)

func TestDescrXML(t *testing.T) {
	const raw = `<?xml version="1.0"?>
<data>
	<sensor name="dev-1" channel="3" type="AT30TSE"/>
	<sensor name="dev-2" channel="3" type="AT30TSE" i2c-addr="0x4d"/>
</data>
`

	type Config struct {
		XMLName xml.Name `xml:"data"`
		Sensors []Descr  `xml:"sensor"`
	}

	var cfg Config
	err := xml.NewDecoder(bytes.NewReader([]byte(raw))).Decode(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	want := []Descr{
		{Name: "dev-1", ChanID: 3, Type: "AT30TSE", I2CAddr: 0},
		{Name: "dev-2", ChanID: 3, Type: "AT30TSE", I2CAddr: 0x4d},
	}
	if !reflect.DeepEqual(want, cfg.Sensors) {
		t.Fatalf("error:\ngot= %v\nwant=%v\n", cfg.Sensors, want)
	}
}
