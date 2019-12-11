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
    if len(os.Args) > 1 {
        LoadConfig(os.Args[1])
    } else {
        LoadConfig("/etc/Sunrise.yaml")
    }

    defer closeHardware()
    defer closeServer()

    initHardware()
    fmt.Println("Hardware initialized")

    initServer()

    //TODO this is only for testing
    //start := time.Now()
    //startTimes[start.Weekday()] = start.Sub(getStartOfDay(start)) + time.Minute / 2
    //wakeUpLength = time.Minute


    waitForAlarms()
    //test()
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
        if !on {
            fmt.Println(now.Weekday())
            alarm := startTimes[now.Weekday()]
            if (alarm >  0) {
                alarmTime := getStartOfDay(now).Add(alarm)
                difference := now.Sub(alarmTime)
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
    router.HandleFunc("/light", DayAlarmHandler)

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
            fmt.Println(err)
            return
        }

        alarm := strings.Split(body.Time, ":")
        hour, err := strconv.Atoi(alarm[0])
        minute, err := strconv.Atoi(alarm[1])
        startTimes[day] = time.Duration(hour)  * time.Hour + time.Duration(minute) * time.Minute
        fmt.Println(day, " set to:", startTimes[day])
    }
}

func LightHandler(response http.ResponseWriter, request *http.Request) {
    if request.Method == "PUT" {
        var body struct {
            On bool
        }

        err := json.NewDecoder(request.Body).Decode(&body)
        if err != nil {
            http.Error(response, err.Error(), http.StatusBadRequest)
            fmt.Println(err)
            return
        }
        fmt.Println("Light set to:", body.On)
        on = body.On
    }
}

func initHardware() {
    setLightBrightness(0)
    if !Settings.Mock {
        err := rpio.Open()
        FatalErrorCheck(err)
        light = rpio.Pin(19)
        light.Mode(rpio.Pwm)
        //Pi supports down to 4688Hz, dimmer supports up to 10kHz
        //Roughly split the difference so everyones in a comfortable range
        light.Freq(125056)
    }
}

//Sets the brightness of the light with 1 being full on
//and 0 being off.
func setLightBrightness(brightness float64) {
    fmt.Println("Brightness to ", brightness)
    var precision uint32 = 64
    cycle := uint32(onBrightness * brightness * float64(precision))
    if !Settings.Mock {
        light.DutyCycle(cycle, precision)
    }
}

func closeHardware() {
    if !Settings.Mock {
        rpio.Close()
    }
}

func closeServer() {
    server.Close()
}
