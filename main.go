package main

import (
    "strings"
    "encoding/json"
    "os"
    "time"
    "github.com/stianeikeland/go-rpio"
    "bufio"
    "fmt"
    "github.com/gorilla/mux"
    "net/http"
    "strconv"
)

var light rpio.Pin
var startTimes = [7]time.Duration{-1, -1, -1, -1, -1, -1, -1}
var wakeUpLength time.Duration = time.Hour
var onBrightness float64 = .25
var minBrightness float64 = .1
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
    startTimes[start.Weekday()] = start.Sub(getStartOfDay(start)) + time.Minute / 2
    wakeUpLength = time.Minute


    light.DutyCycle(0, 32)
    //waitForAlarms()
    test()
}

func test() {
    reader := bufio.NewReader(os.Stdin)
    for i := uint32(0); i < 64; i++ {
        fmt.Println(i)
        light.DutyCycle(i, 64)
        reader.ReadString('\n')
    }
    light.DutyCycle(0, 32)
}

func waitForAlarms() {
    clock := time.NewTicker(time.Second)
    for now := range(clock.C) {
        fmt.Println("now ", now, " on ", on)
        if !on {
            fmt.Println(now.Weekday())
            alarm := startTimes[now.Weekday()]
            fmt.Println("Start time", int(alarm.Hours()), ":", int(alarm.Minutes()) % 60, ":", int(alarm.Seconds()) % 60)
            if (alarm >  0) {
                alarmTime := getStartOfDay(now).Add(alarm)
                difference := now.Sub(alarmTime)
                fmt.Println("difference", difference)
                fmt.Println("difference < wakeUpLength", difference < wakeUpLength)
                if difference > 0 {
                    if difference < wakeUpLength {
                        setLightBrightness(float64(difference) / float64(wakeUpLength))
                    } else {
                        on = true
                        setLightBrightness(1)
                    }
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
    router.HandleFunc("/alarm/{day:[0-6]}", DayAlarmHandler)

    server = http.Server{Addr:"0.0.0.0:5445", Handler: router}

    go func() {
        if err := server.ListenAndServe(); err != nil {
            fmt.Println(err)
        }
    }()
}

func DayAlarmHandler(response http.ResponseWriter, request *http.Request){
    if request.Method == "PUT" {
        day, err := strconv.Atoi(mux.Vars(request)["day"])
        var body struct {
            Time string
        }

        err = json.NewDecoder(request.Body).Decode(&body)
        if err != nil {
            http.Error(response, err.Error(), http.StatusBadRequest)
            return
        }

        alarm := strings.Split(body.Time, ":")
        hour, err := strconv.Atoi(alarm[0])
        minute, err := strconv.Atoi(alarm[1])
        startTimes[day] = time.Duration(hour)  * time.Hour + time.Duration(minute) * time.Minute
        fmt.Println(day, " set to:", startTimes[day])
    }

}

func initHardware() {
    setLightBrightness(0)
    err := rpio.Open()
    if err != nil {
        os.Exit(1)
    }

    light = rpio.Pin(19)
    light.Mode(rpio.Pwm)
    //Pi supports down to 4688Hz, dimmer supports up to 10kHz
    //Roughly split the difference so everyones in a comfortable range
    light.Freq(125056)
}

//Sets the brightness of the light with 1 being full on
//and 0 being off.
func setLightBrightness(brightness float64) {
    var precision uint32 = 64
    cycle := uint32(onBrightness * brightness * float64(precision))
    light.DutyCycle(cycle, precision)
}

func closeHardware() {
    rpio.Close()
}

func closeServer() {
    server.Close()
}
