package promexporter

import (
	"flag"
	"net/http"
	"text/template"

	"log"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sighmon/homekit-enviroplus/internal/types"
)

var (
	temperatureGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "temperature",
			Help: "Temperature measured (*C)",
		},
	)

	pressureGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "pressure",
			Help: "Pressure measured (hPa)",
		},
	)

	humidityGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "humidity",
			Help: "Relative humidity measured (%)",
		},
	)

	oxidisingGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "oxidising",
			Help: "Mostly nitrogen dioxide but could include NO and Hydrogen (Ohms)",
		},
	)

	reducingGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "reducing",
			Help: "Mostly carbon monoxide but could include H2S, Ammonia, Ethanol, Hydrogen, Methane, Propane, Iso-butane (Ohms)",
		},
	)

	nh3Gauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "NH3",
			Help: "mostly Ammonia but could also include Hydrogen, Ethanol, Propane, Iso-butane (Ohms)",
		},
	)

	luxGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "lux",
			Help: "current ambient light level (lux)",
		},
	)

	proximityGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "proximity",
			Help: "proximity, with larger numbers being closer proximity and vice versa",
		},
	)

	pm1Gauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "PM1",
			Help: "Particulate Matter of diameter less than 1 micron. Measured in micrograms per cubic metre (ug/m3)",
		},
	)

	pm25Gauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "PM25",
			Help: "Particulate Matter of diameter less than 2.5 microns. Measured in micrograms per cubic metre (ug/m3)",
		},
	)

	pm10Gauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "PM10",
			Help: "Particulate Matter of diameter less than 10 microns. Measured in micrograms per cubic metre (ug/m3)",
		},
	)

	oxidisingHist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "oxidising_measurements",
		Help:    "Histogram of oxidising measurements",
		Buckets: []float64{0, 10000, 15000, 20000, 25000, 30000, 35000, 40000, 45000, 50000, 55000, 60000, 65000, 70000, 75000, 80000, 85000, 90000, 100000},
	})

	reducingHist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "reducing_measurements",
		Help:    "Histogram of reducing measurements",
		Buckets: []float64{0, 100000, 200000, 300000, 400000, 500000, 600000, 700000, 800000, 900000, 1000000, 1100000, 1200000, 1300000, 1400000, 1500000},
	})

	nh3Hist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "nh3_measurements",
		Help:    "Histogram of nh3 measurements",
		Buckets: []float64{0, 10000, 110000, 210000, 310000, 410000, 510000, 610000, 710000, 810000, 910000, 1010000, 1110000, 1210000, 1310000, 1410000, 1510000, 1610000, 1710000, 1810000, 1910000, 2000000},
	})

	pm1Hist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "pm1_measurements",
		Help:    "Histogram of Particulate Matter of diameter less than 1 micron measurements",
		Buckets: []float64{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100},
	})

	pm25Hist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "pm25_measurements",
		Help:    "Histogram of Particulate Matter of diameter less than 2.5 micron measurements",
		Buckets: []float64{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100},
	})

	pm10Hist = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "pm10_measurements",
		Help:    "Histogram of Particulate Matter of diameter less than 10 micron measurements",
		Buckets: []float64{0, 5, 10, 15, 20, 25, 30, 35, 40, 45, 50, 55, 60, 65, 70, 75, 80, 85, 90, 95, 100},
	})

	index = template.Must(template.New("index").Parse(
		`<!doctype html>
	 <title>Enviro+ Prometheus Exporter</title>
	 <h1>Enviro+ Prometheus Exporter</h1>
	 <a href="/metrics">Metrics</a>
	 <p>
	 `))
)

type Exporter struct {
	address string
}

func New(address string) *Exporter {
	return &Exporter{address: address}
}

func (e *Exporter) Start() {
	flag.Parse()
	log.Printf("Prometheus Exporter starting on port %s\n", e.address)
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		index.Execute(w, "")
	})
	if err := http.ListenAndServe(e.address, nil); err != http.ErrServerClosed {
		panic(err)
	}
}

func (e *Exporter) UpdateReadings(r *types.Readings) {
	temperatureGauge.Set(r.Temperature.Value)
	pressureGauge.Set(r.Pressure.Value)
	humidityGauge.Set(r.Humidity.Value)

	oxidisingGauge.Set(r.Oxidising.Value)
	oxidisingHist.Observe(r.Oxidising.Value)

	reducingGauge.Set(r.Reducing.Value)
	reducingHist.Observe(r.Reducing.Value)

	nh3Gauge.Set(r.Nh3.Value)
	nh3Hist.Observe(r.Nh3.Value)

	luxGauge.Set(r.Lux.Value)
	proximityGauge.Set(r.Proximity.Value)

	pm1Gauge.Set(r.Pm1.Value)
	pm1Hist.Observe(r.Pm1.Value)

	pm25Gauge.Set(r.Pm25.Value)
	pm25Hist.Observe(r.Pm25.Value - r.Pm1.Value)

	pm10Gauge.Set(r.Pm10.Value)
	pm10Hist.Observe(r.Pm10.Value - r.Pm1.Value)
}
