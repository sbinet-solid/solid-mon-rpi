// Copyright 2018 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sensors

import (
	"encoding/xml"
	"fmt"
	"math"
	"strconv"
)

type Descr interface {
	isDescr()
	Descr() *DescrBase
}

type DescrBase struct {
	Name    string
	ChanID  int
	Type    string
	I2CAddr uint8
}

func (d *DescrBase) isDescr()          {}
func (d *DescrBase) Descr() *DescrBase { return d }

func (d *DescrBase) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	var raw struct {
		Name   string `xml:"name,attr"`
		ChanID int    `xml:"channel,attr"`
		Type   string `xml:"type,attr"`
		Addr   string `xml:"i2c-addr,attr"`
	}
	err := dec.DecodeElement(&raw, &start)
	if err != nil {
		return err
	}

	d.Name = raw.Name
	d.ChanID = raw.ChanID
	d.Type = raw.Type
	d.I2CAddr = 0
	if raw.Addr != "" {
		v, err := strconv.ParseUint(raw.Addr, 0, 64)
		if err != nil {
			return err
		}
		if v >= math.MaxUint8 {
			return fmt.Errorf("sensors: address value overflows uint8 (got=%v)", v)
		}
		d.I2CAddr = uint8(v)
	}

	return nil
}

type DescrADC101x struct {
	Base DescrBase
	Vdd  float64

	FullRange int
}

func (d *DescrADC101x) isDescr()          {}
func (d *DescrADC101x) Descr() *DescrBase { return &d.Base }

func (d *DescrADC101x) UnmarshalXML(dec *xml.Decoder, start xml.StartElement) error {
	var raw struct {
		Name   string  `xml:"name,attr"`
		ChanID int     `xml:"channel,attr"`
		Type   string  `xml:"type,attr"`
		Addr   string  `xml:"i2c-addr,attr"`
		Vdd    float64 `xml:"vdd,attr"`
		Frng   int     `xml:"full-range,attr"`
	}
	err := dec.DecodeElement(&raw, &start)
	if err != nil {
		return err
	}

	d.Base.Name = raw.Name
	d.Base.ChanID = raw.ChanID
	d.Base.Type = raw.Type
	d.Base.I2CAddr = 0
	if raw.Addr != "" {
		v, err := strconv.ParseUint(raw.Addr, 0, 64)
		if err != nil {
			return err
		}
		if v >= math.MaxUint8 {
			return fmt.Errorf("sensors: address value overflows uint8 (got=%v)", v)
		}
		d.Base.I2CAddr = uint8(v)
	}
	d.Vdd = raw.Vdd
	if raw.Frng == 0 {
		raw.Frng = 1024
	}
	d.FullRange = raw.Frng

	return nil
}

type DescrAT30TSE struct{ DescrBase }
type DescrHTS221 struct{ DescrBase }
type DescrOnBoard struct{ DescrBase }
