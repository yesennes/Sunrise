# Sunrise

## Building

Requires:

- GNU make
- go tested with 1.13
- ssh and rsync for deploying
- MQTT broker optionally for MQTT use. Tested with Mosquitto 1.5.7
- go-delve/delve optionally for debuging

To get all go dependencies run:

```
go get ./...
```

For development `make` or `make build` will build for the current machine.
`make runlocal` or `make debuglocal` will debug for the local machine, using config.yaml
from the current directory. Make sure to set `mock` to true in the config to disable
hardware if your not on a pi.

## Deploying

`make deploy [TARGET_DIR=/location]` will cross-compile for the pi and rsync it to `/location`.
If location is not given, it will copy it to downloads. `make run TARGET_DIR=/location` will
ssh in and run the program, though the ssh must be kept open to keep it running. Running
as a service will be supported eventually.

The default directory for the config file is `/etc/Sunrise.yaml`, though it can be specified
as the only command line argument.

## MQTT Support

Supports MQTT following the (Homie)[https://homieiot.github.io/] specification. 
The device id is configurable in the config file. Both device id and client id default to 
"sunrise".
You might sets this if your whole family uses Sunrises, to something like `asher-sunrise`.

### Topics

```homie/[deviceid]/light/on```

Turns the machine off it it recieves the 0 bit or the "0" character. Published on when
the device changes its state, i.e. for button. Will cancel an alarm in progress

```
homie/[deviceid]/alarm/[dayOfWeek]
```

Sets the end time of the alarm for the day of week, with 0 being Sunday and 6 being Friday.
Takes a string like "9:00" or "21:00". Set it to -1:00 to disable the alarm for the day

```home/[deviceid]/alarm/wake-up-length```

Time before the alarm to begin the wake up. Takes a string like "1h30m", "15m" or "1h"
