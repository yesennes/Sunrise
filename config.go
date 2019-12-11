package main

import (
    "gopkg.in/yaml.v2"
    "os"
)

type Config struct {
    Mock bool
    Rest struct {
        Enabled bool
        RestPort int
    }
    MQTT struct {
        Enabled bool
        Broker string
        Prefix string
    }
}

var Settings Config

func LoadConfig(location string) {
    file, err := os.Open(location)
    FatalErrorCheck(err)
    err = yaml.NewDecoder(file).Decode(&Settings)
    FatalErrorCheck(err)
}
