package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/brutella/hc/characteristic"
	"github.com/brutella/hc/service"
	"github.com/sighmon/homekit-enviroplus/internal/promexporter"
	"github.com/sighmon/homekit-enviroplus/internal/types"
	"periph.io/x/conn/physic"

	"flag"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/rubiojr/go-enviroplus/bme280"
	"github.com/rubiojr/go-enviroplus/ltr559"
	"github.com/rubiojr/go-enviroplus/mics6814"
	"github.com/rubiojr/go-enviroplus/pms5003"
)

var secondsBetweenReadings time.Duration
var developmentMode bool
var promExporterAddr string
var promExporter bool

func init() {
	flag.StringVar(&promExporterAddr, "prom-exporter-address", ":10006", "Prometheus exporter port number")
	flag.BoolVar(&promExporter, "prom-exporter", false, "Enable the Prometheus exporter")
	flag.DurationVar(&secondsBetweenReadings, "sleep", 1*time.Second, "how many seconds between sensor readings, an int followed by the duration")
	flag.BoolVar(&developmentMode, "dev", false, "turn on development mode to return a random temperature reading, boolean")
	flag.Parse()

	if developmentMode {
		log.Println("Development mode on, ignoring sensor and returning random values...")
	}
}

func calculateAirQuality(pm25 float64, pm10 float64) int {
	// Calculate the Air Quality using the EPA's forumla
	// https://www.epa.vic.gov.au/for-community/monitoring-your-environment/about-epa-airwatch/calculate-air-quality-categories
	// HomeKit	1		2		3		4		5
	// PM2.5	<27		27–62		62–97		97–370		>370
	// PM10		<40		40–80		80–120		120–240		>240
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
	pm25 := characteristic.NewPM2_5Density()
	airQuality.Service.AddCharacteristic(pm25.Characteristic)

	pm10 := characteristic.NewPM10Density()
	airQuality.Service.AddCharacteristic(pm10.Characteristic)

	carbonMonoxide := characteristic.NewCarbonMonoxideLevel()
	airQuality.Service.AddCharacteristic(carbonMonoxide.Characteristic)

	nitrogenDioxide := characteristic.NewNitrogenDioxideDensity()
	airQuality.Service.AddCharacteristic(nitrogenDioxide.Characteristic)

	acc.AddService(airQuality.Service)
	acc.TempSensor.AddLinkedService(airQuality.Service)

	light := service.NewLightSensor()
	acc.AddService(light.Service)
	acc.TempSensor.AddLinkedService(light.Service)

	motion := service.NewMotionSensor()
	acc.AddService(motion.Service)
	acc.TempSensor.AddLinkedService(motion.Service)

	config := hc.Config{
		// Change the default Apple Accessory Pin if you wish
		Pin: "00102003",
		// Port: "12345",
		// StoragePath: "./db",
	}

	var promEx *promexporter.Exporter
	if promExporter {
		promEx = promexporter.New(promExporterAddr)
		go func() {
			promEx.Start()
		}()
	}

	t, err := hc.NewIPTransport(config, acc.Accessory)
	if err != nil {
		log.Fatal(err)
	}

	lp, err := ltr559.New()
	if err != nil {
		log.Fatalf("error initializing ltr559 sensor: %v", err)
	}

	bme, err := bme280.New()
	if err != nil {
		log.Fatalf("error initializing bme280 sensors: %v\n", err)
	}

	pms, err := pms5003.New()
	if err != nil {
		log.Fatal("error initializing pme5003 sensor: %v\n")
	}
	go func() {
		pms.StartReading()
	}()

	mics, err := mics6814.New()
	if err != nil {
		log.Fatal("error initializing mics6814 sensor: %v\n")
	}
	go func() {
		mics.StartReading()
	}()

	// Get the sensor readings every secondsBetweenReadings
	go func() {
		readings := types.Readings{
			Temperature: types.Reading{
				Name:  "temperature",
				Value: 0,
			},
			Humidity: types.Reading{
				Name:  "humidity",
				Value: 0,
			},
			Pressure: types.Reading{
				Name:  "pressure",
				Value: 0,
			},
			Oxidising: types.Reading{
				Name:  "oxidising",
				Value: 0,
			},
			Reducing: types.Reading{
				Name:  "reducing",
				Value: 0,
			},
			Nh3: types.Reading{
				Name:  "NH3",
				Value: 0,
			},
			Lux: types.Reading{
				Name:  "lux",
				Value: 0,
			},
			Proximity: types.Reading{
				Name:  "proximity",
				Value: 0,
			},
			Pm1: types.Reading{
				Name:  "PM1",
				Value: 0,
			},
			Pm25: types.Reading{
				Name:  "PM25",
				Value: 0,
			},
			Pm10: types.Reading{
				Name:  "PM10",
				Value: 0,
			},
		}

		for {
			res, err := bme.Read()
			if err != nil {
				log.Printf("error reading bme280 sensors: %v\n", err)
			} else {
				readings.Temperature.Value = res.Temperature.Celsius()
				readings.Humidity.Value = float64(res.Humidity) / float64(physic.PercentRH)
				readings.Pressure.Value = float64(res.Pressure) / float64(physic.Pascal)
			}

			readings.Lux.Value, err = lp.Lux()
			if err != nil {
				log.Printf("error reading light values: %v", err)
			}
			readings.Proximity.Value, err = lp.Proximity()
			if err != nil {
				log.Printf("error reading proximity values: %v", err)
			}

			// PMS5003
			pmv := pms.LastValue()
			readings.Pm1.Value = float64(pmv.Pm10Std)
			readings.Pm10.Value = float64(pmv.Pm25Std)
			readings.Pm25.Value = float64(pmv.Pm100Std)

			// MICS6814
			micsv := mics.LastValue()
			readings.Oxidising.Value = micsv.Oxidising
			readings.Reducing.Value = micsv.Reducing
			readings.Nh3.Value = micsv.NH3

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
			pm25.SetValue(readings.Pm25.Value)
			pm10.SetValue(readings.Pm10.Value)

			// MICS6814 sensor for carbon monoxide resistance goes down with an increase in ppm (Value 0-100)
			carbonMonoxideValue := (1000 - (readings.Reducing.Value / 1000)) / 100
			carbonMonoxide.SetMaxValue(1000)
			carbonMonoxide.SetValue(carbonMonoxideValue)

			// MICS6814 sensor for nitrogen dioxide resistance goes up with an increase in ug/m3 (Value 0-1000)
			nitrogenDioxideValue := readings.Oxidising.Value / 10000
			nitrogenDioxide.SetValue(nitrogenDioxideValue)

			light.CurrentAmbientLightLevel.SetValue(readings.Lux.Value)

			motion.MotionDetected.SetValue(readings.Proximity.Value > 5)

			log.Println(fmt.Sprintf("Temperature: %.2f°C", readings.Temperature.Value))
			log.Println(fmt.Sprintf("Humidity: %.2f RH", readings.Humidity.Value))
			log.Println(fmt.Sprintf(
				"Air Quality: %d (PM2.5 %f, PM10 %f, CO %f, NO2 %f)",
				calculateAirQuality(readings.Pm25.Value, readings.Pm10.Value),
				readings.Pm25.Value,
				readings.Pm10.Value,
				carbonMonoxideValue,
				nitrogenDioxideValue,
			))
			log.Println(fmt.Sprintf("Light: %.2f lux", readings.Lux.Value))
			log.Println(fmt.Sprintf("Motion: %t (%.2f)", readings.Proximity.Value > 5, readings.Proximity.Value))

			if promExporter {
				promEx.UpdateReadings(&readings)
			}
			// Time between readings
			time.Sleep(secondsBetweenReadings)
		}
	}()

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
