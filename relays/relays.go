package relays

import (
	"fmt"
	"strings"

	rpio "github.com/stianeikeland/go-rpio"
)

// Exported
const PinOff = rpio.High // Seems backward, but a LOW turns the relay on and HIGH turns it off
const PinOn = rpio.Low

// Internal
const pullMode = rpio.PullUp

// Map GPIO pins to relay bank positions
const r1 = 5  // GPIO5
const r2 = 6  // GPIO6
const r3 = 13 // GPIO13
const r4 = 19 // GPIO19
const r5 = 26 // GPIO26
const r6 = 16 // GPIO16
const r7 = 20 // GPIO20
const r8 = 21 // GPIO21

// Map relay bank positions to hot tub functions
const rHeater = r1
const rLight = r3
const rHotBlower = r5
const rColdBlower = r4
const rJets = r2

// Create dictionary of pins
type pin struct {
	name    string
	rpioPin rpio.Pin
}

var pins = []pin{
	{"Heater", rpio.Pin(rHeater)},
	{"Light", rpio.Pin(rLight)},
	{"Jets", rpio.Pin(rJets)},
	{"ColdBlower", rpio.Pin(rColdBlower)},
	{"HotBlower", rpio.Pin(rHotBlower)},
}

//var pins []pin

func log(s string) {
	fmt.Println(s)
}

func Init() {
	if err := rpio.Open(); err != nil {
		panic(err)
	}

	for _, p := range pins {
		rpio.PinMode(p.rpioPin, rpio.Output) // Mode = Output
		rpio.PullMode(p.rpioPin, pullMode)   // PullUp = pin off if floating
		SetPinOff(p.name)
	}

	log("Relays initialized")
}

func CleanUp() {
	// Turn all pins off and unmap GPIO memory range
	AllPinsOff()
	rpio.Close()
}

func AllPinsOff() {
	for _, p := range pins {
		rpio.WritePin(p.rpioPin, PinOff)
	}
}

func FindPin(name string) pin {
	for _, p := range pins {
		if strings.ToLower(p.name) == strings.ToLower(name) {
			return p
			break
		}
	}
	panic("Couldn't find pin " + name)
}

func SetPinOff(name string) {
	FindPin(name).rpioPin.Write(PinOff)
}

func SetPinOn(name string) {
	FindPin(name).rpioPin.Write(PinOn)
}

func SetPin(n string, s bool) {
	if s {
		SetPinOn(n)
	} else {
		SetPinOff(n)
	}
}

func ReadPin(name string) rpio.State {
	return FindPin(name).rpioPin.Read()
}
