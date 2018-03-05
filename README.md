# gohottub
Hot tub controller written in Go!

This is very specific to my personal hot tub configuration:
- Cal Spas circa 1980 8'x8' hot tub w/ crude, very analog control box
- Waterproof DS18b20 digital thermometer
- 8-relay bank spliced into the top side controls (in parallel to existing top side switches)
- Thermometer and relays controlled by GPIO pins of a Raspberry Pi 2 B+ w/ Wifi dongle
- Logging to/reading from three free Thingspeak (https://www.thingspeak.com/) channels
    * One for logging temperature/relay state (e.g., jets, light, blowers)
    * One for writing alerts when they crop up (e.g., shutting down because the thermometer appears to have fallen out of the hot tub)
    * One for a separate app (or curl) to write "commands" (target temperature, jets/light/blower state)
