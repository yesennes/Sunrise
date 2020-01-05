package main

import (
    "gopkg.in/yaml.v2"
    "os"
    "errors"
)

type Config struct {
    Mock bool

    LightPin int
    ButtonPin int
    ZeroCrossPin int

    PullUp bool

    LogLevel string

    Rest struct {
        Enabled bool
        Port int
    }
    Mqtt struct {
        Enabled bool
        Broker string
        DeviceID string
        ClientID string
    }
}


var Settings Config = Config{
    ZeroCrossPin: -1,
    LogLevel: "info",
}

func LoadConfig(location string) {
    file, err := os.Open(location)
    FatalErrorCheck(err)
    decoder := yaml.NewDecoder(file)
    decoder.SetStrict(true)
    err = decoder.Decode(&Settings)
    FatalErrorCheck(err)
    if Settings.LightPin == 0 || Settings.ButtonPin == 0 {
        FatalErrorCheck(errors.New("lightpin or buttonpin unspecified"))
    }

    if Settings.Mqtt.DeviceID == "" {
        Settings.Mqtt.DeviceID = "sunrise"
    }
    if Settings.Mqtt.ClientID == "" {
        Settings.Mqtt.ClientID = "sunrise"
    }
}
