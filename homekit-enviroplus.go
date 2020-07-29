package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"./enviroplus"

	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

var sensorHost string
var sensorPort int
var secondsBetweenReadings time.Duration
var developmentMode bool

func init() {
	flag.StringVar(&sensorHost, "host", "http://0.0.0.0", "sensor host, a string")
	flag.IntVar(&sensorPort, "port", 1006, "sensor port number, an int")
	flag.DurationVar(&secondsBetweenReadings, "sleep", 5*time.Second, "how many seconds between sensor readings, an int followed by the duration")
	flag.BoolVar(&developmentMode, "dev", false, "turn on development mode to return a random temperature reading, boolean")
	flag.Parse()

	if developmentMode == true {
		log.Println("Development mode on, ignoring sensor and returning random values...")
	}
}

func main() {
	info := accessory.Info{
		Name:             "Enviro+",
		SerialNumber:     "PIM486",
		Manufacturer:     "Pimoroni",
		Model:            "Enviro+",
		FirmwareRevision: "1.0.0",
	}

	acc := accessory.NewEnviroPlus(info)
	acc.TemperatureSensor.Name.SetValue = "BME280"
	acc.HumiditySensor.Name.SetValue = "BME280"
	acc.AirQualitySensor.Name.SetValue = "MICS6814 PMS5003"
	acc.LightSensor.Name.SetValue = "LTR-559"

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
		// Match the temperature line in the Prometheus data
		// temperature 17.62804726392642
		re := regexp.MustCompile(`^temperature ([-+]?\d*\.\d+|\d+)`)
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
					rs := re.FindStringSubmatch(line)
					if rs != nil {
						parsedValue, err := strconv.ParseFloat(rs[1], 64)
						if err == nil {
							sensorReading = parsedValue
						}
					}
				}
				scanner = nil
			} else {
				log.Println(err)
			}

			if developmentMode == true {
				// Return a random float between 15 and 30
				sensorReading = 15 + rand.Float64() * (30 - 15)
			}

			// Set the temperature reading on the accessory
			acc.TemperatureSensor.CurrentTemperature.SetValue(sensorReading)
			acc.HumiditySensor.CurrentHumidity.SetValue(sensorReading)
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
