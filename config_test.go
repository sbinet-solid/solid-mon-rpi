// Copyright 2018 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/xml"
	"reflect"
	"testing"

	"github.com/sbinet-solid/solid-mon-rpi/sensors"
)

func TestConfigXML(t *testing.T) {
	const raw = `<?xml version="1.0"?>
<data>
	<sensor name="dev-1" channel="3" type="AT30TSE"/>
	<sensor name="dev-2" channel="3" type="AT30TSE" i2c-addr="0x2d"/>
	<sensor name="dev-3" channel="3" type="ADC101x" i2c-addr="0x3d" vdd="3.2"/>
	<sensor name="dev-4" channel="3" type="ADC101x" i2c-addr="0x4d" vdd="3.2" full-range="256"/>
	<sensor name="dev-5" channel="3" type="HTS221"  i2c-addr="0x5d"/>
	<sensor name="dev-6" channel="3" type="OnBoard" i2c-addr="0x6d"/>
</data>
`

	var cfg Config
	err := xml.NewDecoder(bytes.NewReader([]byte(raw))).Decode(&cfg)
	if err != nil {
		t.Fatal(err)
	}

	want := []sensors.Descr{
		&sensors.DescrAT30TSE{sensors.DescrBase{
			Name: "dev-1", ChanID: 3, Type: "AT30TSE", I2CAddr: 0},
		},
		&sensors.DescrAT30TSE{sensors.DescrBase{
			Name: "dev-2", ChanID: 3, Type: "AT30TSE", I2CAddr: 0x2d},
		},
		&sensors.DescrADC101x{
			Base: sensors.DescrBase{
				Name: "dev-3", ChanID: 3, Type: "ADC101x", I2CAddr: 0x3d,
			},
			Vdd:       3.2,
			FullRange: 1024,
		},
		&sensors.DescrADC101x{
			Base: sensors.DescrBase{
				Name: "dev-4", ChanID: 3, Type: "ADC101x", I2CAddr: 0x4d,
			},
			Vdd:       3.2,
			FullRange: 256,
		},
		&sensors.DescrHTS221{sensors.DescrBase{
			Name: "dev-5", ChanID: 3, Type: "HTS221", I2CAddr: 0x5d},
		},
		&sensors.DescrOnBoard{sensors.DescrBase{
			Name: "dev-6", ChanID: 3, Type: "OnBoard", I2CAddr: 0x6d},
		},
	}
	if !reflect.DeepEqual(want, cfg.Sensors) {
		t.Fatalf("error:\ngot= %v\nwant=%v\n", cfg.Sensors, want)
	}
}
