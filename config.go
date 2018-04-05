// Copyright 2018 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/sbinet-solid/solid-mon-rpi/sensors"
)

type Config struct {
	XMLName xml.Name        `xml:"data"`
	Sensors []sensors.Descr `xml:"sensor"`
	Freq    time.Duration
}

func (cfg *Config) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	cfg.XMLName = start.Name

	tokType := func(attrs []xml.Attr) string {
		for _, attr := range attrs {
			if attr.Name.Local == "type" {
				return attr.Value
			}
		}
		return "???"
	}

	// decode inner elements
	for {
		t, err := dec.Token()
		if err != nil {
			return err
		}
		var descr sensors.Descr
		switch tt := t.(type) {
		case xml.StartElement:
			switch tname := strings.ToLower(tokType(tt.Attr)); tname {
			case "at30tse":
				descr = new(sensors.DescrAT30TSE)
			case "adc101x":
				descr = new(sensors.DescrADC101x)
			case "hts221":
				descr = new(sensors.DescrHTS221)
			case "onboard":
				descr = new(sensors.DescrOnBoard)
			case "bme280":
				descr = new(sensors.DescrBME280)
			default:
				return fmt.Errorf("sensors: invalid type %q", tname)
			}
			err = dec.DecodeElement(descr, &tt)
			if err != nil {
				return err
			}
			cfg.Sensors = append(cfg.Sensors, descr)
		case xml.EndElement:
			if tt == start.End() {
				return nil
			}
		}
	}
	return nil
}
