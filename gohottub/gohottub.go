package main

import (
	"bytes";
	"encoding/csv";
	"fmt";
	_ "io/ioutil";
	"net/http";
	"net/url";
	"strconv";
	"strings";
	"time";
	"github.com/bryandowen/queue";
	"github.com/bryandowen/relays";
	"github.com/bryandowen/thermometer";
//	"github.com/davecgh/go-spew/spew"; // for debugging!
);

// Constants
const Debug bool = true;
const ReadingInterval time.Duration = time.Minute; //TODO: If this changes, we have to update the HeatRate() method on hotTubState!
const TempLimit float64 = 103.5; //degrees (safety limit)
const DefaultTempTarget float64 = 102.0; //default; overwritten by ThingSpeak
const UpperTempWindow float64 = 0.5;
const LowerTempWindow float64 = 0.5;
const HeatingTempDipDelay int64 = 5; //minutes
const SuitableSampleSize int64 = 10; //minutes
const MaxQueueSize = 16;

// TODO: Abstract Thingspeak into a separate package
const ThingspeakUrlBase string = "https://api.thingspeak.com";
const ControlChannelId string = "390551";
const ControlReadKey string = "T7WUFMLHQT7RVOKT";
const ControlWriteKey string = "EEU8VGOHCR1GNGCG"; // only used to clear resets
const DataWriteKey string = "V8EHPSCX44R9FUUU";
const AlertWriteKey string = "ZESCFN0UU1Z4C1PX";
const ControlReadUrl string = ThingspeakUrlBase + "/channels/" + ControlChannelId + "/feeds/last.csv?api_key=" + ControlReadKey + "&results=1";
const ControlWriteUrl string = ThingspeakUrlBase + "/update?api_key=" + ControlWriteKey;
const DataWriteUrl string = ThingspeakUrlBase + "/update?api_key=" + DataWriteKey;
const AlertWriteUrl string = ThingspeakUrlBase + "/update?api_key=" + AlertWriteKey;

func init() {
	relays.Init();
}

func log(s string) {
	fmt.Println(s);
}

func main() {
	// Safety net: clean up relays when we exit!
	// TODO: This ain't working, at least on CTRL+C exit. :-( Need a watchdog process to kill relays if this isn't running
	defer relays.CleanUp();

	log("+++ Startup: Welcome to gohottub 1.2! +++");
	// Set up channels for goroutines
	logTicker := time.NewTicker(ReadingInterval).C;
	commChan := make(chan *tsCommands);
	dataChan := make(chan *tsData);
	alertChan := make(chan *tsAlert);

	// Initialize state
	s := hotTubState{q: queue.New(MaxQueueSize)}; //TODO: Implement .New() pattern (s := hotTubState.New(16);)
	s.SetTemperature(DefaultTempTarget);
	s.isSafe = true;

	// Asynchronous goroutines to interact with Thingspeak
	go readCommands(commChan);
	go logData(dataChan);
	go logAlert(alertChan);

	for {//evar
		// Kick off temperature and command reads
		s.statusMessage = "";
		s.cycleCounter++;
		if s.HeaterState() && s.cycleCounter == HeatingTempDipDelay {
			// We give a grace period after the heater kicks on because the temperature
			// immediately drops due to de-stratification in the tub, and the water in
			// the heater plumbing cooling quicker than that in the tub
			s.q.Drain();
			log("Drained temperature queue after " + strconv.FormatInt(HeatingTempDipDelay, 10) + " readings");
		}
		commChan <- nil // prompt command goroutine to read commands
		s.temperature = thermometer.GetTemperatureF();
		dataChan <- s.GetDataChan();
		log("    DS18b20 temperature: " + strconv.FormatFloat(s.temperature, 'f', 2, 64) + " (negative = cooling)");
		s.q.Enqueue(s.temperature);

		// Shut down if temp is too high!
		if s.HeaterState() && s.temperature > TempLimit {
			// TODO: Should this go in a separate goroutine?
			log(">> Uh-oh! Temp > 104; shutting down!");
			shutdown(&s, alertChan, fmt.Sprintf("Temperature exceeds limit (%.2f)", TempLimit));
		}

		// Alert/shutdown if temp rate is out of whack
		heatRate := s.HeatRate();
		log("              Heat rate: " + strconv.FormatFloat(heatRate, 'f', 2, 64) + " (positive = overtemp)");

		if s.cycleCounter > SuitableSampleSize {
			switch s.HeaterState() {
				case true: //heating
					log("                  Mode: Heating");
					switch {
						case heatRate >= 9.0:
							log(">> Heating greater than 9deg/hr. : This is insanely fast");
							shutdown(&s, alertChan, "Heating over 9deg/hr.!! (Not sure what this even means.)");
						case heatRate >= 5:
							log(">> Heating greater than 5deg/hr. : Heating normally");
							s.statusMessage = "Heating normally";
						case heatRate >= 1:
							log(">> Heating greater than 1deg/hr. : Cover likely open (heating slowly)");
							s.statusMessage = "Heating slowly; cover likely open";
						default:
							log(">> Heating less than 1deg/hr. : Heating too slow, shutting down (check thermometer and over-temp switch)");
							shutdown(&s, alertChan, "Heating too slowly (over-temp button? thermometer in tub?)");
					}
				//}
				case false: //cooling
					log("                  Mode: Cooling");
					switch {
						case heatRate > 2.5:
							log(">> Heating greater than 2.5deg/hr. : Should be cooling, but is heating");
							s.statusMessage = "Should be cooling, but is heating"; // TODO: shutdown?
						case heatRate > -2:
							log(">> Cooling less than 2deg/hr. : Cooling normally");
							s.statusMessage = "Cooling normally";
						case heatRate > -15:
							log(">> Cooling less than 15deg/hr. : Cover likely open");
							s.statusMessage = "Cover likely open (cooling quickly)";
						default:
							log(">> Cooling over 15deg/hr.!! : Thermometer likely fallen out");
							shutdown(&s, alertChan, "Cooling too fast (thermometer in tub?)");
					}
				//}
			}
		} else {
			log(">> Skipping alert/shutdown checks for another " + strconv.FormatInt(SuitableSampleSize - s.cycleCounter, 10) + " readings.");
		}

		// Process commands read from Thingspeak
		c := <-commChan;
		s.SetTargetTemperature(c.TargetTemperature);
		s.SetJetsState(c.TargetJets);
		s.SetLightState(c.TargetLight);
		s.SetColdBlowerState(c.TargetColdBlower);
		s.SetHotBlowerState(c.TargetHotBlower);

		// TODO: Check for alert reset signal on Command channel
		//       - Set back to isSafe: true
		//       - Write back to Command channel to clear the flag

		// Regulate temperature
		loopError := s.TargetTemperature() - s.Temperature(); // We want to control this toward zero (+/- window)
		if s.isSafe {// Only regulate temp/relays if in Safe mode!
			log("             Loop error: " + strconv.FormatFloat(loopError, 'f', 2, 64));
			if s.HeaterState() {
				if loopError < (0-UpperTempWindow) { // e.g., drops to -0.6 w/ window of 0.5
					turnHeaterOff(&s);
				}
			} else {//cooling
				if loopError > LowerTempWindow {
					turnHeaterOn(&s);
				}
			}
		} else {
			log("!!!!! Not regulating temperature; hot tub is in Safe Mode");
		}

		// Now wait for next clock tick
		// TODO: Do this in a select{} so we're not "sleeping" and can be responsive to (e.g.) overtemp
		log(s.PrettyPrint());
		log("============================================================");

		<-logTicker;
	}
}

func turnHeaterOn(s *hotTubState) {
	relays.SetPinOn("heater"); // de facto backing "variable" for s.HeaterState()
	s.q.Drain();
	s.cycleCounter = 0;
	log(">> Turned heater on");
	log("*** s.Heater: " + strconv.FormatBool(s.HeaterState()) + "***");
}

func turnHeaterOff(s *hotTubState) {
	relays.SetPinOff("heater");
	s.q.Drain();
	s.cycleCounter = 0;
	log(">> Turned heater off");
}

func shutdown(s *hotTubState, alertChan chan *tsAlert, msg string) {
	log(">> Shutting down");
	turnHeaterOff(s);
	s.isSafe = false;
	s.statusMessage = msg;
	alertChan <- &tsAlert{Message: msg};
}

// TODO: Factor all goroutines into another package
// TODO: Move to a model where a goroutine provides a temperature feed to (at least) a couple
//   others, which watch (a) whether to turn on/off the heater, (b) whether we're heating or
//   cooling too fast (on a long queue) and (c) whether the thermometer has likely fallen out
//   of the tub (on a short queue) (e.g., 3 readings, and if the spread is > 1 degree?

func callHttp(u string) (string, int) {
	// Make call
	resp, err := http.Get(u); if err != nil { panic(err); }

	// Retrieve status and body from response
	status := resp.StatusCode;
	defer resp.Body.Close();
	buf := new(bytes.Buffer);
	buf.ReadFrom(resp.Body);

	// React to 429 (rate limit exceeded); TODO: Something more graceful, like a retry
	if status == http.StatusTooManyRequests {
		panic("Exceeded Thingspeak rate limit");
	}

	return buf.String(), status;
}

type tsCommands struct {
	TargetTemperature float64 `field1`;
	// There is no TargetHeater since that's regulated by the temperature setting.
	TargetJets bool `field2`;
	TargetLight bool `field3`;
	TargetColdBlower bool `field4`;
	TargetHotBlower bool `field5`;
}
func (c *tsCommands) PrettyPrint() string {
        s := "tsCommands:\n";
        s += "    TargetTemperature  : " + strconv.FormatFloat(c.TargetTemperature, 'f', 2, 64) + "\n";
        s += "    TargetJets         : " + strconv.FormatBool(c.TargetJets) + "\n";
        s += "    TargetLight        : " + strconv.FormatBool(c.TargetLight) + "\n";
        s += "    TargetColdBlower   : " + strconv.FormatBool(c.TargetColdBlower) + "\n";
        s += "    TargetHotBlower    : " + strconv.FormatBool(c.TargetHotBlower) + "\n";
        return s;
}

// goroutine
func readCommands(commChan chan *tsCommands) {
	u := ControlReadUrl;
	for {//evar
		<- commChan // Wait for trigger to read commands from Thingspeak

		// Make Thingspeak call
		body, _ := callHttp(u);

		// Parse response
		r := csv.NewReader(strings.NewReader(body));
		records, err := r.ReadAll(); if err != nil { panic(err); }
		if len(records) != 2 || len(records[0]) < 8 {
			panic("Invalid commands response from ThingSpeak (record too short or narrow):" + body);
		}

		// Assimilate response into channel struct
		cc := tsCommands{};
		cc.TargetTemperature, err = strconv.ParseFloat(records[1][2], 64);
		cc.TargetJets, err = strconv.ParseBool(records[1][3]);
		cc.TargetLight, err = strconv.ParseBool(records[1][4]);
		cc.TargetColdBlower, err = strconv.ParseBool(records[1][5]);
		cc.TargetHotBlower, err = strconv.ParseBool(records[1][6]);

		log(cc.PrettyPrint());

		commChan <- &cc; // Push commands to channel
	}
} // TODO: Comb through ThingSpeak fields, make sure they're what we want (no targetTemp, etc.) & reconcile here

type tsData struct {
	Temperature float64 `field1`;
	Heater bool `field2`
	Jets bool `field3`;
	Light bool `field4`;
	ColdBlower bool `field5`;
	HotBlower bool `field6`;
	StatusMessage string `field7`;
	HeatRate float64 `field8`;
}
func (t *tsData) PrettyPrint() string {
	s := "tsData:\n";
	s += "     Temperature   : " + strconv.FormatFloat(t.Temperature, 'f', 2, 64) + "\n";
	s += "     Heater        : " + strconv.FormatBool(t.Heater) + "\n";
	s += "     Jets          : " + strconv.FormatBool(t.Jets) + "\n";
	s += "     Light         : " + strconv.FormatBool(t.Light) + "\n";
	s += "     ColdBlower    : " + strconv.FormatBool(t.ColdBlower) + "\n";
	s += "     HotBlower     : " + strconv.FormatBool(t.HotBlower) + "\n";
	s += "     StatusMessage : " + t.StatusMessage + "\n";
	s += "     HeatRate      : " + strconv.FormatFloat(t.HeatRate, 'f', 2, 64) + "\n";
	return s;
}

// goroutine
func logData(dataChan chan *tsData) {
	for {//evar
		d := <- dataChan // Wait for trigger to log temperature to Thingspeak
		u := fmt.Sprintf("%s&field1=%s&field2=%s&field3=%s&field4=%s&field5=%s&field6=%s&field7=%s&field8=%s", DataWriteUrl, url.QueryEscape(strconv.FormatFloat(d.Temperature, 'f', 2, 64)), boolToNumString(d.Heater), boolToNumString(d.Jets), boolToNumString(d.Light), boolToNumString(d.ColdBlower), boolToNumString(d.HotBlower), url.QueryEscape(d.StatusMessage), url.QueryEscape(strconv.FormatFloat(d.HeatRate, 'f', 2, 64)));
		callHttp(u); // Write-only call
	}
}

func boolToNumString(b bool) string {
	if b {
		return "1";
	} else {
		return "0";
	}
}

type tsAlert struct {
	Message string `field1`;
}

// goroutine
func logAlert(alertChan chan *tsAlert) {
	for {//evar
		a := <- alertChan // Wait for trigger to log temperature to Thingspeak
		u := fmt.Sprintf("%s&field1=%s", AlertWriteUrl, url.QueryEscape(a.Message));
		callHttp(u); // Write-only call
	}
}

type hotTubState struct {
	temperature float64;
	targetTemperature float64;
	q *queue.Queue;
	cycleCounter int64;
	statusMessage string;
	isSafe bool;
}
func (h *hotTubState) PrettyPrint() string {
	s := "hotTubState:\n";
	s += "     targetTemperature : " + strconv.FormatFloat(h.targetTemperature, 'f', 2, 64) + "\n";
	s += "     temperature       : " + strconv.FormatFloat(h.temperature, 'f', 2, 64) + "\n";
	s += "     isSafe            : " + strconv.FormatBool(h.isSafe) + "\n";
	s += "     HeaterState()     : " + strconv.FormatBool(h.HeaterState()) + "\n";
	s += "     JetsState()       : " + strconv.FormatBool(h.JetsState()) + "\n";
	s += "     LightState()      : " + strconv.FormatBool(h.LightState()) + "\n";
	s += "     ColdBlowerState() : " + strconv.FormatBool(h.ColdBlowerState()) + "\n";
	s += "     HotBlowerState()  : " + strconv.FormatBool(h.HotBlowerState()) + "\n";
	s += "     statusMessage     : " + h.statusMessage + "\n";
	s += "     cycleCounter      : " + strconv.FormatInt(h.cycleCounter, 10) + "\n";
	s += "     q                 : [length: " + strconv.FormatInt(int64(len(h.q.Inspect())), 10) + "/" + strconv.FormatInt(MaxQueueSize, 10) + "]\n";
	for i, qn := range h.q.Inspect() {
		s += "                         [" + strconv.FormatInt(int64(i), 10) + "]: " + strconv.FormatFloat(qn, 'f', 2, 64) + "\n";
	}
	return s;
}
func (h *hotTubState) Temperature() float64 {
        return h.temperature;
}
func (h *hotTubState) SetTemperature(t float64) {
	h.temperature = t;
}
func (h *hotTubState) TargetTemperature() float64 {
        return h.targetTemperature;
}
func (h *hotTubState) SetTargetTemperature(t float64) {
        h.targetTemperature = t;
}
func (h *hotTubState) HeaterState() bool {
        return relays.ReadPin("heater") == relays.PinOn;
}
//func (h *hotTubState) SetHeaterState(s bool) { // Regulated by temperature routine
//        relays.SetPin("heater", s);
//}
func (h *hotTubState) JetsState() bool {
        return relays.ReadPin("jets") == relays.PinOn;
}
func (h *hotTubState) SetJetsState(s bool) {
	relays.SetPin("jets", s);
}
func (h *hotTubState) LightState() bool {
        return relays.ReadPin("light") == relays.PinOn;
}
func (h *hotTubState) SetLightState(s bool) {
	relays.SetPin("light", s);
}
func (h *hotTubState) ColdBlowerState() bool {
        return relays.ReadPin("coldblower") == relays.PinOn;
}
func (h *hotTubState) SetColdBlowerState(s bool) {
	relays.SetPin("coldblower", s);
}
func (h *hotTubState) HotBlowerState() bool {
        return relays.ReadPin("hotblower") == relays.PinOn;
}
func (h *hotTubState) SetHotBlowerState(s bool) {
	relays.SetPin("hotblower", s);
}
func (h *hotTubState) HeatRate() float64 { // method on hotTubState
	q := h.q.Inspect();
	if len(q) < 2 {
		return 0.0;
	} else {
		// TODO: Is there a better way to calculate rate? (Especially on short data windows)
		// TODO: If we need an average: var total float64; for _, value := range q { total += value; } ...
		return (q[len(q) - 1] - q[0]) * (60.0/float64(len(q))); // TODO: Fix; assumes 1-minute read interval
	}
}
func (h *hotTubState) GetDataChan() *tsData {
	return &tsData {
		Temperature: h.temperature,
		Heater: relays.ReadPin("heater") == relays.PinOn,
		Jets: relays.ReadPin("jets") == relays.PinOn,
		Light: relays.ReadPin("light") == relays.PinOn,
		ColdBlower: relays.ReadPin("coldblower") == relays.PinOn,
		HotBlower: relays.ReadPin("hotblower") == relays.PinOn,
		StatusMessage: h.statusMessage,
		HeatRate: h.HeatRate(),
	}
}
