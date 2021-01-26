package types

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
