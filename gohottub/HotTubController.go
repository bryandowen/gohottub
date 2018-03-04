package main

import (
	"bytes";
	"encoding/csv";
	"fmt";
	_ "io/ioutil";
	"net/http";
	"net/url";
	"os";
	"strconv";
	"strings";
	"time";
	log "github.com/sirupsen/logrus";
	"github.com/bryandowen/queue";
	"github.com/bryandowen/thermometer";
	_ "github.com/davecgh/go-spew/spew"; // for debugging!
);

// Constants
const Debug bool = true;
const ReadingDelay int = 60 //seconds; TODO: If this changes, we have to update the TempRate() method on hotTubState!
const TempLimit float64 = 103.5 //degrees (safety limit)
const TempTarget float64 = 102.0 //default; overwritten by ThingSpeak
const UpperTempWindow float64 = 0.5;
const LowerTempWindow float64 = 0.5;
const HeatingTempDipDelay int = 5 //minutes
const SuitableSampleSize int = 10 //minutes
const ThingspeakUrlBase string = "https://api.thingspeak.com";
const DataWriteKey string = "V8EHPSCX44R9FUUU";
const DataWriteUrl string = ThingspeakUrlBase + "/update?api_key=" + DataWriteKey;
const ControlChannelId string = "390551";
const ControlReadKey string = "T7WUFMLHQT7RVOKT";
const ControlReadUrl string = ThingspeakUrlBase + "/channels/" + ControlChannelId + "/feeds/last.csv?api_key=" + ControlReadKey + "&results=1";
const ControlWriteKey string = "EEU8VGOHCR1GNGCG"; // only used to clear resets
const ControlWriteUrl string = ThingspeakUrlBase + "/update?api_key=" + ControlWriteKey;
const AlertWriteKey string = "ZESCFN0UU1Z4C1PX";
const AlertWriteUrl string = ThingspeakUrlBase + "/update?api_key=" + AlertWriteKey;

func init() {
	log.SetFormatter(&log.JSONFormatter{});
	log.SetOutput(os.Stdout);
	log.SetLevel(log.DebugLevel);
	// Log levels: Debug, Info, Warn, Error, Fatal, Panic
	// Log.WithFields(log.Fields{"key": "value", "key": value,})
}

func main() {
	// TODO: Initialize relays
	logTicker := time.NewTicker(time.Minute).C;
	commChan := make(chan *tsCommands);
	dataChan := make(chan *tsData);
	alertChan := make(chan *tsAlert);

	s := hotTubState{q: queue.New(16)}; //TODO: Implement .New() pattern (s := hotTubState.New(16);)
	s.targetTemperature = TempTarget; // Default until commands are read from Thingspeak

	// Asynchronous goroutines to interact with Thingspeak
	go readCommands(commChan);
	go logData(dataChan);
	go logAlert(alertChan);

	for {//evar
		// Kick off temperature and command reads
		s.cycleCounter++;
		commChan <- nil // prompt command goroutine to read commands
		log.Debug("cycleCounter: " + strconv.FormatInt(int64(s.cycleCounter), 10));
		s.temperature = thermometer.GetTemperatureF();
		log.Info("Read temperature: " + strconv.FormatFloat(s.temperature, 'f', -1, 64));
		s.q.Enqueue(s.temperature);

		// Shut down if temp is too high!
		if s.temperature > TempLimit {
			// TODO: Should this go in a separate goroutine?
			log.Warn("Uh-oh! Temp > 104; shutting down!");
			shutdown(s, alertChan, fmt.Sprintf("Temperature exceeds limit (%.2f)", TempLimit));
		}

		// Alert/shutdown if temp rate is out of whack
		tempRate := s.TempRate();
		switch s.isHeaterOn {
			case true: //heating
				log.Debug("Mode: Heating");
				switch {
					case tempRate >= 9.0:
						log.Debug("tempRate >= 9.0; this is insanely fast");
						shutdown(s, alertChan, "Heat rate exceeds 9deg/hr.");
					case tempRate >= 5:
						log.Debug("tempRate >= 5; heating normally");
						s.alertMessage = "Heating normally";
					case tempRate >= 1:
						log.Debug("tempRate >= 1; cover likely open (heating slowly)");
						s.alertMessage = "Cover likely open (heating slowly)";
					default:
						log.Debug("tempRate < 1; heating too slow, shutting down (check thermometer and over-temp switch)");
						shutdown(s, alertChan, "Heating too slowly (over-temp button? thermometer in tub?)");
				}
			//}
			case false: //cooling
				log.Debug("Mode: Cooling");
				switch {
					case tempRate > 2.5:
						log.Debug("tempRate > 2.5; should be heating, but is cooling");
						s.alertMessage = "Should be cooling, but is heating"; // TODO: shutdown?
					case tempRate > -2:
						log.Debug("tempRate > -2; cooling normally");
						s.alertMessage = "Cooling normally";
					case tempRate > -10:
						log.Debug("tempRate > -10; cover likely open");
						s.alertMessage = "Cover likely open (cooling quickly)";
					default:
						log.Debug("Cooling mega-fast! Thermometer likely fallen out");
						shutdown(s, alertChan, "Cooling too fast (thermometer in tub?)");
				}
			//}
		}

		// Process commands read from Thingspeak
		log.Debug("Blocking on commChan");
		commands := <-commChan; // 5. Read commands from read channel
		log.Debug("commChan received!");
		s.targetTemperature = commands.TargetTemperature; // 6. Set tub state according to commands
		s.targetJetsState = commands.TargetJets;
		s.targetLightState = commands.TargetLight;
		s.targetColdBlowerState = commands.TargetColdBlower;
		s.targetHotBlowerState = commands.TargetHotBlower;

		// TODO: Set relay states for jets, light, coldblower, hotblower

		// Regulate temperature
		loopError := s.targetTemperature - s.temperature; // We want to control this toward zero (+/- window)
		log.Info("Loop error: " + strconv.FormatFloat(loopError, 'f', -1, 64));
		if s.isHeaterOn {
			if loopError < (0-UpperTempWindow) { // e.g., drops to -0.6 w/ window of 0.5
				turnHeaterOff(s);
			}
		} else {//cooling
			if loopError > LowerTempWindow {
				turnHeaterOn(s);
			}
		}

		// Now sleep to start over in 1 minute
		// TODO: Do this in a select{} so we're not "sleeping" and can be responsive to (e.g.) overtemp
		log.Debug("Blocking on logTicker...");
		<-logTicker;
		log.Debug("Received logTicker!");
	}
}

func turnHeaterOn(s hotTubState) {
	s.isHeaterOn = true;
	// TODO: Turn on heater relay
}

func turnHeaterOff(s hotTubState) {
	s.isHeaterOn = false;
	// TODO: Turn off heater relay
}

func shutdown(s hotTubState, alertChan chan *tsAlert, msg string) {
	log.Warn("Shutting down");
	s.isSafe = false;
	alertChan <- &tsAlert{Message: msg};
}

func callHttp(uri string) (string, int) {
	// Make call
	resp, err := http.Get(uri); if err != nil { panic(err); }

	// Retrieve status and body from response
	status := resp.StatusCode;
	defer resp.Body.Close();
	buf := new(bytes.Buffer);
	buf.ReadFrom(resp.Body);

	// React to 429 (rate limit exceeded); TODO: Something more graceful, like a retry
	log.Debug(fmt.Sprintf("[callHttp]: Called %s\nResponse Code: %s\nResponse Body:\n%s", uri, status, buf.String()));
	if status == http.StatusTooManyRequests {
		panic("Exceeded Thingspeak rate limit");
	}

	return buf.String(), status;
}

// goroutine
func readCommands(commChan chan *tsCommands) {
	u := CONTROL_READ_URL;
	for {//evar
		log.Debug("Blocking on commChan...");
		<- commChan // Wait for trigger to read commands from Thingspeak
		log.Debug("commChan received!");

		// Make Thingspeak call
		body, _ := callHttp(u);

		// Parse response
		r := csv.NewReader(strings.NewReader(body));
		records, err := r.ReadAll(); if err != nil { panic(err); }
		if len(records) != 2 || len(records[0]) < 8 {
			panic("Invalid commands response from ThingSpeak (record too short or narrow)");
		}

		// Assimilate response into channel struct
		cc := tsCommands{};
		cc.TargetTemperature, err = strconv.ParseFloat(records[1][2], 64);
		cc.TargetJets, err = strconv.ParseBool(records[1][3]);
		cc.TargetLight, err = strconv.ParseBool(records[1][4]);
		cc.TargetColdBlower, err = strconv.ParseBool(records[1][5]);
		cc.TargetHotBlower, err = strconv.ParseBool(records[1][6]);
		log.WithFields(log.Fields{"TargetTemperature": cc.TargetTemperature, "TargetJets": cc.TargetJets, "TargetLight": cc.TargetLight, "TargetColdBlower": cc.TargetColdBlower, "TargetHotBlower": cc.TargetHotBlower,}).Debug("Temperature retrieved");

		commChan <- &cc; // Push commands to channel
	}
} // TODO: Comb through ThingSpeak fields, make sure they're what we want (no targetTemp, etc.) & reconcile here

// goroutine
func logData(dataChan chan *tsData) {
	for {//evar
		log.Debug("Blocking on dataChan...");
		d := <- dataChan // Wait for trigger to log temperature to Thingspeak
		log.Debug("Received dataChan!");

		u := fmt.Sprintf("%s&field1=%s&field2=%s&field3=%s&field4=%s&field5=%s&field6=%s&field7=%s&field8=%s", DATA_WRITE_URL, url.QueryEscape(strconv.FormatFloat(d.Temperature, 'f', -1, 64)), url.QueryEscape(strconv.FormatBool(d.Jets)), url.QueryEscape(strconv.FormatBool(d.Light)), url.QueryEscape(strconv.FormatBool(d.ColdBlower)), url.QueryEscape(strconv.FormatBool(d.HotBlower)), url.QueryEscape(d.AlertMessage), url.QueryEscape(strconv.FormatFloat(d.HeatRate, 'f', -1, 64)));
		callHttp(u); // Write-only call
	}
}

// goroutine
func logAlert(alertChan chan *tsAlert) {
	for {//evar
		log.Debug("Blocking on alertChan...");
		a := <- alertChan // Wait for trigger to log temperature to Thingspeak
		log.Debug("Received alertChan!");

		u := fmt.Sprintf("%s&field1=%s", ALERT_WRITE_URL, url.QueryEscape(a.Message));
		callHttp(u); // Write-only call
	}
}

type hotTubState struct {
	// actuals
	temperature float64;
	q *queue.Queue;
	cycleCounter int;
	isHeaterOn bool;
	alertMessage string;
	isSafe bool;
	// targets
	targetTemperature float64;
	targetJetsState bool;
	targetLightState bool;
	targetColdBlowerState bool;
	targetHotBlowerState bool;
}

func (h *hotTubState) TempRate() float64 { // method on hotTubState
	q := h.q.Inspect();
	if len(q) < 2 {
		return 0.0;
	} else {
		// TODO: Is there a better way to calculate rate? (Especially on short data windows)
		// TODO: If we need an average: var total float64; for _, value := range q { total += value; } ...
		return (q[len(q) - 1] - q[0]) * (60.0/float64(len(q))); // TODO: Fix; assumes 1-minute read interval
	}
}

type tsData struct {
	Temperature float64 `field1`;
	Jets bool `field2`;
	Light bool `field3`;
	ColdBlower bool `field4`;
	HotBlower bool `field5`;
	AlertMessage string `field6`;
	HeatRate float64 `field7`;
	// TODO: Right now there's a field 8 being used for heatRate; remember to shift these in ThingSpeak
}

type tsCommands struct {
	TargetTemperature float64 `field1`;
	TargetJets bool `field2`;
	TargetLight bool `field3`;
	TargetHeater bool `field4`;
	TargetColdBlower bool `field5`;
	TargetHotBlower bool `field6`;
}

type tsAlert struct {
	Message string `field1`;
}
