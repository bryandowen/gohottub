# gohottub
Hot tub controller written in Go!

This is very specific to my personal hot tub configuration:
- Cal Spas circa 1990 hot tub w/ crude, very analog control box
- Waterproof DS18b20 digital thermometer (4.7kOhm resistor across DC+ and data lines)
- Relay bank spliced into the top side controls (in series with existing top side switches)
  (turn on all top-side controls unless you want to override this controller and turn off)
- Thermometer and relays controlled by GPIO pins of a Raspberry Pi 3 B+
- Logging to/reading from three free Thingspeak channels (https://www.thingspeak.com/)
    * One for logging temperature/relay state (e.g., jets, light, blowers)
    * One for writing alerts when they crop up (e.g., shutting down because the thermometer appears to have fallen out of the hot tub)
    * One for a separate app (or curl) to write "commands" (target temperature, jets/light/blower state)

## INSTALLATION

To run (just scripts+binary [and code]):
```
mkdir ~/go/src/github.com/bryandowen
cd ~/go/src/github.com/bryandowen
sudo apt-get install git # if git not installed already
git clone https://github.com/bryandowen/gohottub
sudo mkdir /var/log/gohottub
sudo chown ${USER}:${USER} /var/log/gohottub
sudo echo "dtoverlay=w1-gpio" >> /boot/firmware/config.txt
sudo modprobe wire
sudo modprobe w1-gpio
sudo modprobe w1-therm
sudo reboot now
cd go/src/github.com/bryandowen/gohottub
./start & # beware: hard-coded directory may not be correct
```

#### If this script errors out:
```
fg 1
<CTRL+C>
```
... then edit start to correct directories. Sorry, I'll clean this up later.

## CONFIGURE TO RUN AUTOMATICALLY ON BOOT
```
cd scripts
./install
```

Now logs will be rotated weekly, and you can use ```systemctl``` to manage the service:
```
systemctl status gohottub.service       # See if it's running properly
sudo systemctl restart gohottub.service # bounce
sudo systemctl stop gohottub.service    # stop
sudo systemctl start gohottub.service   # start
sudo systemctl disable gohottub.service # Stop it from automatically running on boot
sudo systemctl enable gohottub.service  # Set it to start running automatically again
```

## MONITORING
You'll want to be able to check up on the service to make sure it's running properly
```
tail -f /var/log/gohottub/gohottub.log # Watch current output
systemctl status gohottub.service      # Make sure services hasn't died
```

## FOR DEVELOPMENT
```
cd ..
mkdir stianeikeland
cd stianeikeland
git clone https://github.com/stianeikeland/go-rpio
cd ..
mkdir yryz
cd yryz
git clone https://github.com/yryz/ds18b20
cd ..
cd bryandowen/gohottub/gohottub
sudo apt-get update # if Go not installed already
sudo apt-get install golang # if Go not installed already
go build ./gohottub.go
```
