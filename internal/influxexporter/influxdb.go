package influxexporter

import (
	"errors"
	"fmt"
	"os"
	"time"

	influxdb "github.com/influxdata/influxdb-client-go/v2"
	"github.com/sighmon/homekit-enviroplus/internal/types"
)

const INTERVAL = 1 * time.Minute

type Exporter struct {
	client                    influxdb.Client
	url, token, orgID, bucket string
	sensorName                string
}

func New(sensorName string) (*Exporter, error) {
	e := &Exporter{sensorName: sensorName}
	e.url = os.Getenv("INFLUXDB_URL")
	if e.url == "" {
		return nil, errors.New("invalid influxdb URL")
	}
	e.token = os.Getenv("INFLUXDB_TOKEN")
	if e.token == "" {
		return nil, errors.New("invalid influxdb token")
	}
	e.orgID = os.Getenv("INFLUXDB_ORG_ID")
	if e.orgID == "" {
		return nil, errors.New("invalid influxdb ORG ID")
	}
	e.bucket = os.Getenv("INFLUXDB_BUCKET")
	if e.bucket == "" {
		return nil, errors.New("invalid influxdb bucket")
	}

	e.client = influxdb.NewClient(e.url, e.token)

	return e, nil
}

func (e *Exporter) UpdateReadings(r *types.Readings) {
	e.postToInflux(r.Humidity.Name, r.Humidity.Value)
	e.postToInflux(r.Oxidising.Name, r.Oxidising.Value)
	e.postToInflux(r.Reducing.Name, r.Reducing.Value)
	e.postToInflux(r.Pm1.Name, r.Pm1.Value)
	e.postToInflux(r.Pm25.Name, r.Pm25.Value)
	e.postToInflux(r.Pm10.Name, r.Pm10.Value)
	e.postToInflux(r.Pressure.Name, r.Pressure.Value)
	e.postToInflux(r.Lux.Name, r.Lux.Value)
	e.postToInflux(r.Nh3.Name, r.Nh3.Value)
	e.postToInflux(r.Proximity.Name, r.Proximity.Value)
	e.postToInflux(r.Temperature.Name, r.Temperature.Value)
}

func (e *Exporter) postToInflux(name string, value float64) {
	payload := e.sensorName + fmt.Sprintf(" %s=%f", name, value)

	writeAPI := e.client.WriteAPI(e.orgID, e.bucket)
	writeAPI.WriteRecord(payload)
	writeAPI.Flush()
}
