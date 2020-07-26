package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"

	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

var sensorHost string
var sensorPort int
var secondsBetweenReadings time.Duration

func init() {
	flag.StringVar(&sensorHost, "host", "http://0.0.0.0", "sensor host, a string")
	flag.IntVar(&sensorPort, "port", 1006, "sensor port number, an int")
	flag.DurationVar(&secondsBetweenReadings, "sleep", 5*time.Second, "how many seconds between sensor readings, an int followed by the duration")
	flag.Parse()
}

func main() {
	info := accessory.Info{
		Name:             "Enviro+",
		SerialNumber:     "PIM486",
		Manufacturer:     "Pimoroni",
		Model:            "Enviro+",
		FirmwareRevision: "1.0.0",
	}

	acc := accessory.NewTemperatureSensor(
		info,
		0.0,             // Initial value
		-40.0,           // Min sensor value
		85.0,            // Max sensor value
		0.1,             // Step value
	)

	config := hc.Config{
		// Change the default Apple Accessory Pin if you wish
		Pin: "00102003",
		// Port: "12345",
		// StoragePath: "./db",
	}

	t, err := hc.NewIPTransport(config, acc.Accessory)
	if err != nil {
		log.Fatal(err)
	}

	// Get the sensor readings every secondsBetweenReadings
	go func() {
		for {
			// Get readings from the Prometheus exporter
			sensorReading := 0.0
			resp, err := http.Get(fmt.Sprintf("%s:%d", sensorHost, sensorPort))
			if err == nil {
				defer resp.Body.Close()
				scanner := bufio.NewScanner(resp.Body)
				for scanner.Scan() {
					line := scanner.Text()
					// Parse the temperature reading
					re := regexp.MustCompile(`^temperature ([-+]?\d*\.\d+|\d+)`)
					rs := re.FindStringSubmatch(line)
					if rs != nil {
						parsedValue, err := strconv.ParseFloat(rs[1], 64)
						if err == nil {
							sensorReading = parsedValue
						}
					}
				}
			} else {
				log.Println(err)
			}

			// Set the temperature reading on the accessory
			acc.TempSensor.CurrentTemperature.SetValue(sensorReading)
			log.Println(fmt.Sprintf("Temperature: %f°C", sensorReading))

			// Time between readings
			time.Sleep(secondsBetweenReadings)
		}
	}()

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
