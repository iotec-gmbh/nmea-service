package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	nmea "github.com/adrianmo/go-nmea"
	"github.com/tarm/serial"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	yearOffset    = 2000            // offset in years for GSP Signal
	serialTimeout = 5 * time.Second // Timeout for the serial connection
)

// data is the struct that holds all relevant GPS information.
// lowercase variables are ignored during json.Marshal
type data struct {
	m            *sync.Mutex
	update       time.Time
	Timestamp    time.Time
	Longitude    float64
	Latitude     float64
	LongitudeGPS string
	LatitudeGPS  string
	LongitudeDMS string
	LatitudeDMS  string
	Altitude     float64
	Satellites   int64
	Age          time.Duration
}

var (
	// Command line options parsed via kingpin. These are pointers.
	verbose  = kingpin.Flag("verbose", "Enable verbose mode.").Bool()
	tty      = kingpin.Flag("tty", "Serial Connection.").Default("/dev/ttyUSB0").String()
	baudrate = kingpin.Flag("baudrate", "Baudrate of the Serial Connection.").Default("115200").Int()
	host     = kingpin.Flag("host", "Host to listen.").Default("localhost").String()
	port     = kingpin.Flag("port", "Port to listen on.").Default("54321").Int()
	// d is the instance of data that is updated from the GPS sensor and which is marshaled and send via HTTP
	d = data{
		m: &sync.Mutex{},
	}
)

// updateGPS updates 'd' with the information from the GPS sensor.
func updateGPS(r io.Reader) {
	// Use a buffered reader. We do not want to read byte-wise and look for newlines.
	reader := bufio.NewReader(r)

	// Loop for parsing
	for {
		// Read line
		sentence, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error while reading from serial, %v", err)
			continue
		}

		// Strip \r\n from the sentence
		sentence = strings.TrimSuffix(strings.TrimSuffix(sentence, "\n"), "\r")

		// Verbose output
		if *verbose {
			log.Printf("Raw Sentence: %v\n", sentence)
		}

		// Parse sentence via nmea parser
		s, err := nmea.Parse(sentence)
		if err != nil {
			log.Printf("Error while parsing '%v', %v", sentence, err)
			continue
		}

		// Different NMEA types needs to be handled differently
		switch m := s.(type) {
		// We collect the timestamp from the GPRMC and also set the last updated here
		case nmea.GPRMC:
			d.m.Lock()
			d.Timestamp = time.Date(
				yearOffset+m.Date.YY, time.Month(m.Date.MM), m.Date.DD,
				m.Time.Hour, m.Time.Minute, m.Time.Second, m.Time.Millisecond,
				time.UTC)
			d.update = time.Now()
			d.m.Unlock()
			if *verbose {
				log.Printf("New time %v\n", d.Timestamp)
			}
		// FROM GGA we collect the GPS location information
		case nmea.GPGGA:
			d.m.Lock()
			d.Altitude = m.Altitude
			d.Longitude = m.Longitude
			d.Latitude = m.Latitude
			d.LatitudeGPS = nmea.FormatGPS(m.Latitude)
			d.LongitudeGPS = nmea.FormatGPS(m.Longitude)
			d.LatitudeDMS = nmea.FormatDMS(m.Latitude)
			d.LongitudeDMS = nmea.FormatDMS(m.Longitude)
			d.Satellites = m.NumSatellites
			d.m.Unlock()
			if *verbose {
				log.Printf("Latitude: %v\n", m.Latitude)
				log.Printf("Longitude: %v\n", m.Longitude)
				log.Printf("Altitude: %v\n", m.Altitude)

				log.Printf("Satellites: %v\n", m.NumSatellites)
			}
		// All remaining types are skipped
		default:
			if *verbose {
				log.Printf("Skipping %T\n", s)
			}
		}
	}
}

// HTTP Handler to send 'd' as JSON
func handler(w http.ResponseWriter, r *http.Request) {
	// Set age as time duration from last time GPRMC was parsed and now
	d.m.Lock()
	d.Age = time.Since(d.update)
	// JSONify
	js, err := json.Marshal(d)
	d.m.Unlock()
	if err != nil {
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// mainWithError contains main loop but can return errors
func mainWithError() error {
	// Parse command line
	kingpin.Parse()
	if *verbose {
		log.Println("Running in verbose mode.")
		log.Printf("Using tty %v\n", *tty)
		log.Printf("Using baudrate %v\n", *baudrate)
		log.Printf("Using host %v\n", *host)
		log.Printf("Using port %v\n", *port)
	}

	// Open Serial Connection
	c := &serial.Config{Name: *tty, Baud: *baudrate, ReadTimeout: serialTimeout}
	s, err := serial.OpenPort(c)
	if err != nil {
		return err
	}

	// Run updateGPS to keep 'd' up to date in go routine
	go updateGPS(s)

	// Start HTTP Server
	http.HandleFunc("/", handler)
	return http.ListenAndServe(fmt.Sprintf("%v:%v", *host, *port), nil)
}

// main calls mainWithError and log error
func main() {
	err := mainWithError()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
