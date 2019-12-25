package main

import (
    "gopkg.in/yaml.v2"
    "os"
    "fmt"
)

type Config struct {
    Mock bool

    LightPin int
    ButtonPin int
    ZeroCrossPin int

    PullUp bool

    Rest struct {
        Enabled bool
        Port int
    }
    Mqtt struct {
        Enabled bool
        Broker string
        Prefix string
        ClientID string
    }
}

var Settings Config

func LoadConfig(location string) {
    file, err := os.Open(location)
    FatalErrorCheck(err)
    fmt.Println("File read")
    decoder := yaml.NewDecoder(file)
    decoder.SetStrict(true)
    err = decoder.Decode(&Settings)
    FatalErrorCheck(err)
    fmt.Println("Config read")
}
