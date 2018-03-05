package thermometer;

import (
        "github.com/yryz/ds18b20"
);

func GetTemperatureC() float64 {
	sensor := getSensor();
	return getTemperatureCelsius(sensor);
}

func GetTemperatureF() float64 {
	return celsiusToFahrenheit(GetTemperatureC());
}

func getSensor() string {
	sensors, err := ds18b20.Sensors();
	if err != nil {
		panic(err);
	}
	return sensors[0];
}

func getTemperatureCelsius(sensor string) float64 {
	t, err := ds18b20.Temperature(sensor);
	if err != nil {
		panic(err);
	}
	return t;
}

func celsiusToFahrenheit(tempC float64) float64 {
	tf := tempC * 9/5 + 32;
	tf *= 100;
	itf := int(tf);
	return float64(itf)/float64(100);
}
