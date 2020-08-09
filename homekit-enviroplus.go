package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/service"

	"bufio"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"reflect"
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

func calculateAirQuality(pm25 float64, pm10 float64) int {
	// Calculate the Air Quality using the EPA's forumla
	// https://www.epa.vic.gov.au/for-community/monitoring-your-environment/about-epa-airwatch/calculate-air-quality-categories
	// HomeKit	1		2		3		4		5
	// PM2.5	<27		27–62	62–97	97–370	>370
	// PM10		<40		40–80	80–120	120–240	>240
	pm25Quality := 0
	pm10Quality := 0
	switch {
	case pm25 < 27:
		pm25Quality = 1
	case pm25 >= 27 && pm25 <= 61:
		pm25Quality = 2
	case pm25 >= 62 && pm25 <= 96:
		pm25Quality = 3
	case pm25 >= 97 && pm25 <= 369:
		pm25Quality = 4
	case pm25 >= 370:
		pm25Quality = 5
	}

	switch {
	case pm10 < 27:
		pm10Quality = 1
	case pm10 >= 27 && pm10 <= 61:
		pm10Quality = 2
	case pm10 >= 62 && pm10 <= 96:
		pm10Quality = 3
	case pm10 >= 97 && pm10 <= 369:
		pm10Quality = 4
	case pm10 >= 370:
		pm10Quality = 5
	}

	if pm25Quality > pm10Quality {
		return pm25Quality
	}

	return pm10Quality
}

func main() {
	info := accessory.Info{
		Name:             "Enviro+",
		SerialNumber:     "PIM486",
		Manufacturer:     "Pimoroni",
		Model:            "Enviro+",
		FirmwareRevision: "1.0.0",
	}

	// acc := accessory.NewEnviroPlus(info)
	// acc.TemperatureSensor.Name.SetValue = "BME280"
	// acc.HumiditySensor.Name.SetValue = "BME280"
	// acc.AirQualitySensor.Name.SetValue = "MICS6814 PMS5003"
	// acc.LightSensor.Name.SetValue = "LTR-559"

	acc := accessory.NewTemperatureSensor(
		info,
		0.0,   // Initial value
		-40.0, // Min sensor value
		85.0,  // Max sensor value
		0.1,   // Step value
	)

	humidity := service.NewHumiditySensor()
	acc.AddService(humidity.Service)
	acc.TempSensor.AddLinkedService(humidity.Service)

	airQuality := service.NewAirQualitySensor()
	acc.AddService(airQuality.Service)
	acc.TempSensor.AddLinkedService(airQuality.Service)

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
		type Reading struct {
			Name  string
			Value float64
		}

		type Readings struct {
			Temperature Reading
			Humidity    Reading
			Pressure    Reading
			Oxidising   Reading
			Reducing    Reading
			Nh3         Reading
			Lux         Reading
			Proximity   Reading
			Pm1         Reading
			Pm25        Reading
			Pm10        Reading
		}

		readings := Readings{
			Temperature: Reading{
				Name:  "temperature",
				Value: 0,
			},
			Humidity: Reading{
				Name:  "humidity",
				Value: 0,
			},
			Pressure: Reading{
				Name:  "pressure",
				Value: 0,
			},
			Oxidising: Reading{
				Name:  "oxidising",
				Value: 0,
			},
			Reducing: Reading{
				Name:  "reducing",
				Value: 0,
			},
			Nh3: Reading{
				Name:  "NH3",
				Value: 0,
			},
			Lux: Reading{
				Name:  "lux",
				Value: 0,
			},
			Proximity: Reading{
				Name:  "proximity",
				Value: 0,
			},
			Pm1: Reading{
				Name:  "PM1",
				Value: 0,
			},
			Pm25: Reading{
				Name:  "PM25",
				Value: 0,
			},
			Pm10: Reading{
				Name:  "PM10",
				Value: 0,
			},
		}
		values := reflect.ValueOf(readings)

		for {
			// Get readings from the Prometheus exporter
			resp, err := http.Get(fmt.Sprintf("%s:%d", sensorHost, sensorPort))
			if err == nil {
				defer resp.Body.Close()
				scanner := bufio.NewScanner(resp.Body)
				for scanner.Scan() {
					line := scanner.Text()
					// Parse the readings
					for i := 0; i < values.NumField(); i++ {
						fieldname := values.Field(i).Interface().(Reading).Name
						regexString := fmt.Sprintf("^%s", fieldname) + ` ([-+]?\d*\.\d+|\d+)`
						re := regexp.MustCompile(regexString)
						rs := re.FindStringSubmatch(line)
						if rs != nil {
							parsedValue, err := strconv.ParseFloat(rs[1], 64)
							if err == nil {
								if developmentMode {
									println(fmt.Sprintf("%s %f", fieldname, parsedValue))
								}

								// TOFIX: Work out how to set the Value of the Reading... this causes a panic
								// reflect.ValueOf(readings).FieldByName(strings.ToTitle(fieldname)).FieldByName("Value").SetFloat(parsedValue)
								// For now use a switch statement
								switch fieldname {
								case "temperature":
									readings.Temperature.Value = parsedValue
								case "humidity":
									readings.Humidity.Value = parsedValue
								case "pressure":
									readings.Pressure.Value = parsedValue
								case "oxidising":
									readings.Oxidising.Value = parsedValue
								case "reducing":
									readings.Reducing.Value = parsedValue
								case "NH3":
									readings.Nh3.Value = parsedValue
								case "lux":
									readings.Lux.Value = parsedValue
								case "proximity":
									readings.Proximity.Value = parsedValue
								case "PM1":
									readings.Pm1.Value = parsedValue
								case "PM25":
									readings.Pm25.Value = parsedValue
								case "PM10":
									readings.Pm10.Value = parsedValue
								}
							}
						}
					}
				}
				scanner = nil
			} else {
				log.Println(err)
			}

			if developmentMode {
				// Return a random float between 15 and 30
				readings.Temperature.Value = 15 + rand.Float64()*(30-15)
			}

			// Set the sensor readings
			acc.TempSensor.CurrentTemperature.SetValue(readings.Temperature.Value)
			acc.TempSensor.CurrentTemperature.SetStepValue(0.1)
			humidity.CurrentRelativeHumidity.SetValue(readings.Humidity.Value)
			humidity.CurrentRelativeHumidity.SetStepValue(0.1)
			airQuality.AirQuality.SetValue(calculateAirQuality(readings.Pm25.Value, readings.Pm10.Value))
			log.Println(fmt.Sprintf("Temperature: %f°C", readings.Temperature.Value))
			log.Println(fmt.Sprintf("Humidity: %f RH", readings.Humidity.Value))
			log.Println(fmt.Sprintf("Air Quality: %d (PM2.5 %f, PM10 %f)", calculateAirQuality(readings.Pm25.Value, readings.Pm10.Value), readings.Pm25.Value, readings.Pm10.Value))

			// Time between readings
			time.Sleep(secondsBetweenReadings)
		}
	}()

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
