// Copyright 2017 The solid-mon-rpi Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"sort"
	"time"

	"golang.org/x/net/websocket"

	"github.com/go-daq/smbus"
	"github.com/gonum/plot/palette/brewer"
	"github.com/sbinet-solid/solid-mon-rpi/sensors"
)

func main() {
	var (
		addr    = flag.String("addr", ":8080", "[ip]:port for TCP server")
		busID   = flag.Int("bus-id", 0x1, "SMBus ID number (/dev/i2c-[ID]")
		busAddr = flag.Int("bus-addr", 0x70, "SMBus address to read/write")
		freq    = flag.Duration("freq", 2*time.Second, "data polling interval")
		cfgFlag = flag.String("cfg", "", "path to an XML configuration file for sensors")
	)

	flag.Parse()

	log.SetFlags(0)
	log.SetPrefix("solid-mon-rpi ")

	log.Printf("starting up web-server on: %v\n", *addr)
	srv, err := newServer(*addr, *freq, *busID, *busAddr)
	if err != nil {
		log.Fatalf("error starting server: %v", err)
	}

	if *cfgFlag != "" {
		var cfg Config
		f, err := os.Open(*cfgFlag)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		err = xml.NewDecoder(f).Decode(&cfg)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("cfg: %+v\n", cfg.Sensors)
		srv.bus.descr = cfg.Sensors

		set := make(map[string]int)
		for _, descr := range srv.bus.descr {
			set[descr.Name] = 1
		}
		var labels []string
		for k := range set {
			labels = append(labels, k)
		}
		sort.Strings(labels)
		p, err := brewer.GetPalette(brewer.TypeAny, "Dark2", len(labels))
		if err != nil {
			log.Fatal(err)
		}
		colors := p.Colors()
		for i, label := range labels {
			col := colors[i%len(colors)]
			plotColors[label] = col
		}
	}

	http.Handle("/", srv)
	http.Handle("/data", websocket.Handler(srv.dataHandler))
	http.HandleFunc("/echo", srv.wrap(srv.echoHandler))

	err = http.ListenAndServe(srv.addr, nil)
	if err != nil {
		srv.quit <- 1
		log.Fatalf("error running server: %v", err)
	}
}

type Config struct {
	XMLName xml.Name        `xml:"data"`
	Sensors []sensors.Descr `xml:"sensor"`
	Freq    time.Duration
}

type server struct {
	addr string
	freq time.Duration
	quit chan int

	bus struct {
		id    int
		addr  uint8
		data  chan sensors.Sensors
		descr []sensors.Descr
	}

	tmpl    *template.Template
	dataReg registry // clients interested in sensors data
	plots   chan Plots
	echo    chan sensors.Sensors
}

func newServer(addr string, freq time.Duration, busID, busAddr int) (*server, error) {
	if addr == "" {
		addr = getHostIP() + ":80"
	}

	srv := &server{
		addr:    addr,
		freq:    freq,
		quit:    make(chan int),
		dataReg: newRegistry(),
		tmpl:    template.Must(template.New("fcs").Parse(indexTmpl)),
		plots:   make(chan Plots),
		echo:    make(chan sensors.Sensors),
	}
	srv.bus.id = busID
	srv.bus.addr = uint8(busAddr)
	srv.bus.data = make(chan sensors.Sensors)

	conn, err := smbus.Open(srv.bus.id, srv.bus.addr)
	if err != nil {
		return nil, fmt.Errorf(
			"error opening SMBus connection (id=%d addr=0x%x): %v",
			srv.bus.id,
			srv.bus.addr,
			err,
		)
	}
	go srv.run(conn)

	return srv, nil
}

func (srv *server) Freq() float64 {
	return 1 / srv.freq.Seconds()
}

func (srv *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	srv.wrap(srv.rootHandler)(w, r)
}

func (srv *server) rootHandler(w http.ResponseWriter, r *http.Request) error {
	return srv.tmpl.Execute(w, srv)
}

func (srv *server) wrap(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err != nil {
			log.Printf("error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

func (srv *server) echoHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return fmt.Errorf("invalid HTTP request (got=%v, want=%v)", r.Method, http.MethodGet)
	}
	timeout := time.NewTimer(2 * srv.freq)
	defer timeout.Stop()
	select {
	case <-timeout.C:
		return fmt.Errorf("timeout retrieving data from board")
	case data := <-srv.echo:
		w.Header().Set("Content-Type", "application/json")
		err := json.NewEncoder(w).Encode(data)
		if err != nil {
			return err
		}
	}
	return nil
}

func (srv *server) dataHandler(ws *websocket.Conn) {
	log.Printf("new client...")
	c := &client{
		srv:   srv,
		reg:   &srv.dataReg,
		datac: make(chan []byte, 256),
		ws:    ws,
	}
	c.reg.register <- c
	defer c.Release()

	c.run()
}

func (srv *server) run(bus *smbus.Conn) {
	go srv.daq(bus)
	go srv.mon()
	for {
		select {
		case c := <-srv.dataReg.register:
			log.Printf("client registering [%v]...", c.ws.LocalAddr())
			srv.dataReg.clients[c] = true

		case c := <-srv.dataReg.unregister:
			if _, ok := srv.dataReg.clients[c]; ok {
				delete(srv.dataReg.clients, c)
				close(c.datac)
				log.Printf(
					"client disconnected [%v]\n",
					c.ws.LocalAddr(),
				)
			}

		case plots := <-srv.plots:
			if len(srv.dataReg.clients) == 0 {
				// no client connected
				continue
			}
			dataBuf := new(bytes.Buffer)
			err := json.NewEncoder(dataBuf).Encode(&plots)
			if err != nil {
				log.Printf("error marshalling data: %v\n", err)
				continue
			}
			for c := range srv.dataReg.clients {
				select {
				case c.datac <- dataBuf.Bytes():
				default:
					close(c.datac)
					delete(srv.dataReg.clients, c)
				}
			}
		}
	}
}

func (srv *server) daq(bus *smbus.Conn) {
	defer bus.Close()

	tick := time.NewTicker(srv.freq)
	defer tick.Stop()

	i := 0
	for range tick.C {
		data, err := srv.fetchData(bus)
		if err != nil {
			log.Printf("error fetching data: %v\n", err)
			continue
		}

		i++
		if i%10 == 0 {
			log.Printf("daq: %+v\n", data)
		}
		select {
		case srv.bus.data <- data:
		default:
			// nobody is listening
			// drop it on the floor
		}
	}
}

func (srv *server) mon() {
	trendTick := time.NewTicker(time.Minute)
	defer trendTick.Stop()

	table := newNtuple()
	trend := newNtuple()

	first := true
	var data sensors.Sensors
	for {
		select {
		case data = <-srv.bus.data:
			table.add(data)
			if first {
				first = false
				trend.add(data)
			}
			psFast, err := newControlPlots(table.data)
			if err != nil {
				log.Printf("error creating monitoring plots: %v", err)
				continue
			}
			psSlow, err := newControlPlots(trend.data)
			if err != nil {
				log.Printf("error creating (trend) monitoring plots: %v", err)
				continue
			}
			ps := Plots{
				update: time.Now().UTC(),
				plots:  psFast,
				trends: psSlow,
				data:   data,
			}
			select {
			case srv.plots <- ps:
			default:
				// nobody is listening
			}
		case <-trendTick.C:
			trend.add(data)

		case srv.echo <- data:
		}
	}
}

type ntuple struct {
	data []sensors.Sensors
}

func newNtuple() *ntuple {
	return &ntuple{
		data: make([]sensors.Sensors, 0, 2048),
	}
}

func (nt *ntuple) add(data sensors.Sensors) {
	if len(nt.data) == cap(nt.data) {
		i := len(nt.data)
		copy(nt.data[:i/2], nt.data[i/2:])
		nt.data = nt.data[:i/2]
	}
	nt.data = append(nt.data, data)
}

func (srv *server) fetchData(bus *smbus.Conn) (sensors.Sensors, error) {
	data, err := sensors.New(bus, srv.bus.addr, srv.bus.descr)
	if err != nil {
		return data, err
	}

	return data, nil
}

func getHostIP() string {
	host, err := os.Hostname()
	if err != nil {
		log.Fatalf("could not retrieve hostname: %v\n", err)
	}

	addrs, err := net.LookupIP(host)
	if err != nil {
		log.Fatalf("could not lookup hostname IP: %v\n", err)
	}

	for _, addr := range addrs {
		ipv4 := addr.To4()
		if ipv4 == nil {
			continue
		}
		return ipv4.String()
	}

	log.Fatalf("could not infer host IP")
	return ""
}
