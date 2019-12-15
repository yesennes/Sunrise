# Sunrise

## Building

Requires:

- GNU make
- go tested with 1.13
- ssh and scp for deploying
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

`make deploy` will cross-compile for the pi and scp it to downloads. `make run` will
ssh in and run the program, though the ssh must be kept open to keep it running. Running
as a service will be supported eventually.

The default directory for the config file is `/etc/Sunrise.yaml`, though it can be specified
as the only command line argument.
