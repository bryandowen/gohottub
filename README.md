# gohottub
Hot tub controller written in Go!

This is very specific to my personal hot tub configuration:
- Cal Spas circa 1980 8'x8' hot tub w/ crude, very analog control box
- Waterproof DS18b20 digital thermometer
- 8-relay bank spliced into the top side controls (in series with existing top side switches)
  (turn on all top-side controls unless you want to override and turn off)
- Thermometer and relays controlled by GPIO pins of a Raspberry Pi 2 B+ w/ Wifi dongle
- Logging to/reading from three free Thingspeak (https://www.thingspeak.com/) channels
    * One for logging temperature/relay state (e.g., jets, light, blowers)
    * One for writing alerts when they crop up (e.g., shutting down because the thermometer appears to have fallen out of the hot tub)
    * One for a separate app (or curl) to write "commands" (target temperature, jets/light/blower state)

INSTALLATION

To run (just scripts+binary [and code]):
- $ mkdir ~/go/src/github.com/bryandowen
- $ cd ~/go/src/github.com/bryandowen
- $ git clone https://github.com/bryandowen/gohottub
- $ sudo mkdir /var/log/gohottub
- $ sudo chown pi:pi /var/log/gohottub
- $ sudo echo "dtoverlay=w1-gpio" >> /boot/config.txt
- $ sudo modprobe wire
- $ sudo modprobe w1-gpio
- $ sudo modprobe w1-therm
- $ sudo reboot now
- $ cd go/src/github.com/bryandowen/gohottub
- $ ./start &

TODO: Making it run automatically on reboot

For development:
- $ cd ..
- $ mkdir stianeikeland
- $ cd stianeikeland
- $ git clone https://github.com/stianeikeland/go-rpio
- $ cd ..
- $ mkdir yryz
- $ cd yryz
- $ git clone https://github.com/yryz/ds18b20
- $ cd ..
- $ cd bryandowen/gohottub
- $ go build ./gohottub.go
- $ sudo apt-get update
- $ sudo apt-get install golang # installs 1.7, latest is 1.9 :-\
