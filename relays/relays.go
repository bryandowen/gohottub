package relays

import (
	"github.com/stianeikeland/go-rpio"
)

const HEATER_PIN = 5;
const LIGHT_PIN = 6;
const JETS_PIN = 26;
const COLD_BLOWER_PIN = 19;
const HOT_BLOWER_PIN = 13;
const PIN_OFF = rpio.High; // Seems backward, but a LOW turns the relay on
const PIN_ON = rpio.Low;   // and a HIGH turns the relay off


func Init() {
	pinHeater := rpio.Pin(HEATER_PIN);
	pinHeater.PullUp();
	pinHeater.Output();
	pinLight := rpio.Pin(LIGHT_PIN);
	pinLight.PullUp();
	pinLight.Output();
	pinJets := rpio.Pin(JETS_PIN);
	pinLight.PullUp();
	pinLight.Output();
	pinColdBlower := rpio.Pin(COLD_BLOWER_PIN);
	pinColdBlower.PullUp();
	pinColdBlower.Output();
	pinHotBlower := rpio.Pin(HOT_BLOWER_PIN);
	pinHotBlower.PullUp();
	pinHotBlower.Output();
	HeaterOff();
	:ightOff();
	JetsOff();
	ColdBlowerOff();
	HotBlowerOff();
}

func HeaterOn() {
	pinHeater.Write(PIN_ON);
}

func HeaterOff() {
	pinHeater.Write(PIN_OFF);
}

func LightOn() {
	pinLight.Write(PIN_ON);
}

func LightOff() {
	pinLight.Write(PIN_OFF);
}

func JetsOn() {
	pinJets.Write(PIN_ON);
}

func JetsOff() {
	pinJets.Write(PIN_OFF);
}

func ColdBlowerOn() {
	pinColdBlower.Write(PIN_ON);
}

func ColdBlowerOff() {
	pinColdBlower.Write(PIN_OFF);
}

func HotBlowerOn() {
	pinHotBlower.Write(PIN_ON);
}

func HotBlowerOff() {
	pinHotBlower.Write(PIN_OFF);
}

func AllPinsOff() {
	heaterOff();
	lightOff();
	jetsOff();
	coldBlowerOff();
	hotBlowerOff();
}

func readPin
func ReadAllPins() []pinState {
	
}

type pinState struct {
	name string;
	pinNumber int;
	state bool;
}
