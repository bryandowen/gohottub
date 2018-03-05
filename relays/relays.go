package relays

import (
	"github.com/stianeikeland/go-rpio";
	"strings";
)

const PinOff = rpio.High; // Seems backward, but a LOW turns the relay on and HIGH turns it off
const PinOn = rpio.Low;
const PullMode = rpio.PullUp;

type pin struct {
	name string;
	rpioPin rpio.Pin;
}

var pins []pin;

func Init() {
	if err := rpio.Open(); err != nil { panic(err);	}

	pins = []pin {
		{"Heater", rpio.Pin(5)},
		{"Light", rpio.Pin(6)},
		{"Jets", rpio.Pin(26)},
		{"ColdBlower", rpio.Pin(19)},
		{"HotBlower", rpio.Pin(13)},
	}

	for _, p := range pins {
		rpio.PinMode(p.rpioPin, rpio.Output);      // Mode = Output
		rpio.PullMode(p.rpioPin, PullMode); // PullUp = pin off if floating
		SetPinOff(p.name);
	}
}

func CleanUp() {
	// Turn all pins off and unmap GPIO memory range
	AllPinsOff();
	rpio.Close();
}

func AllPinsOff() {
	for _, p := range pins {
		rpio.WritePin(p.rpioPin, PinOff);
	}
}

func FindPin(name string) pin {
	for _, p := range pins {
		if strings.ToLower(p.name) == strings.ToLower(name) {
			return p;
			break;
		}
	}
	panic("Couldn't find pin " + name);
}

func SetPinOff(name string) {
	FindPin(name).rpioPin.Write(PinOff);
}

func SetPinOn(name string) {
	FindPin(name).rpioPin.Write(PinOn);
}

func SetPin(n string, s bool) {
	if s {
		SetPinOn(n);
	} else {
		SetPinOff(n);
	}
}

func ReadPin(name string) rpio.State {
	return FindPin(name).rpioPin.Read();
}
