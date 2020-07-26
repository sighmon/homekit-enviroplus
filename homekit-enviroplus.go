package main

import (
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"

	"fmt"
	"log"
	"math/rand"
	"time"
)

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
		-40.0,           // Min value (sensor: -40)
		85.0,            // Max value (sensor: 85)
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

	// Get the sensor readings every n seconds
	go func() {
		for {
			// TODO: get readings from Prometheus exporter
			// For now let's return a random float between 15 and 30
			sensor_reading := 15 + rand.Float64() * (30 - 15)

			// Set reading
			acc.TempSensor.CurrentTemperature.SetValue(sensor_reading)
			log.Println(fmt.Sprintf("Temperature: %f°C", sensor_reading))
			time.Sleep(5 * time.Second)
		}
	}()

	hc.OnTermination(func() {
		<-t.Stop()
	})

	t.Start()
}
