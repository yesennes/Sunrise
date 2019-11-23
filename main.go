package main

import (
        "os"
        "time"
        "github.com/stianeikeland/go-rpio"
        //"bufio"
        "fmt"
        "github.com/gorilla/mux"
        "net/http"
)

var light rpio.Pin
var startTimes = [7]time.Duration{-1, -1, -1, -1, -1, -1, -1}
var wakeUpLength time.Duration = time.Hour
var onBrightness float64 = 1
var on bool = false

var server http.Server

func main() {
        defer closeHardware()
        defer closeServer()

        initHardware()
        fmt.Println("Hardware initialized")

        initServer()

        start := time.Now()
        //TODO this is only for testing
        startTimes[start.Weekday()] = start.Sub(getStartOfDay(start)) + time.Minute
        wakeUpLength = time.Minute * 2


        setLightBrightness(1)
        for {
        }
}

func waitForAlarms() {
        clock := time.NewTicker(time.Second)
        for now := range(clock.C) {
                fmt.Println(now)
                if !on {
                        fmt.Println(now.Weekday())
                        alarm := startTimes[now.Weekday()]
                        fmt.Println("Start time", int(alarm.Hours()), ":", int(alarm.Minutes()) % 60, ":", int(alarm.Seconds()) % 60)
                        if (alarm >  0) {
                                alarmTime := getStartOfDay(now).Add(alarm)
                                difference := now.Sub(alarmTime)
                                fmt.Println("difference", difference)
                                fmt.Println("difference < wakeUpLength", difference < wakeUpLength)
                                if difference < wakeUpLength && difference > 0{
                                        setLightBrightness(float64(difference) / float64(wakeUpLength))
                                } else {
                                        on = true
                                        setLightBrightness(1)
                                }
                        }
                }
        }
}

func getStartOfDay(t time.Time) time.Time {
        year, month, day := t.Date()
        return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func initServer() {
        router := mux.NewRouter()

        server = http.Server{Addr:"5445", Handler: router}
}

func initHardware() {
        err := rpio.Open()
        if err != nil {
                os.Exit(1)
        }

        light := rpio.Pin(19)
        light.Mode(rpio.Pwm)
        //Pi supports down to 4688Hz, dimmer supports up to 10kHz
        //Roughly split the difference so everyones in a comfortable range
        light.Freq(7500)
        setLightBrightness(0)
}

//Sets the brightness of the light with 1 being full on
//and 0 being off.
func setLightBrightness(brightness float64) {
        fmt.Println("brightness", brightness)
        fmt.Println("Brightness to ", uint32(onBrightness * brightness * 32), "/32")
        light.DutyCycle(uint32(onBrightness * brightness * 32), 32)
}

func closeHardware() {
        rpio.Close()
}

func closeServer() {
        server.Close()
}
