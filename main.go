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
    "gobot.io/x/gobot/platforms/mqtt"
)

var startTimes = [7]time.Duration{-1, -1, -1, -1, -1, -1, -1}
var wakeUpLength time.Duration = time.Hour
var onBrightness float64 = .25
var minBrightness float64 = .1

var on bool = false
var alarmInProgress bool = false
var currentBrightness float64 = -1

var light rpio.Pin
var mqttAdaptor *mqtt.Adaptor
var server http.Server

func main() {
    if len(os.Args) > 1 {
        LoadConfig(os.Args[1])
    } else {
        LoadConfig("/etc/Sunrise.yaml")
    }

    defer closeHardware()

    initHardware()
    fmt.Println("Hardware initialized")

    if Settings.Rest.Enabled {
        initServer()
        defer closeServer()
    }
    if Settings.Mqtt.Enabled {
        initMQTT()
        defer mqttAdaptor.Disconnect()
    }

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
            alarm := startTimes[now.Weekday()]
            if (alarm < 0) {
                setLightBrightness(0)
            } else {
                alarmTime := getStartOfDay(now).Add(alarm)
                difference := now.Sub(alarmTime)
                if difference > 0 {
                    if difference < wakeUpLength {
                        setLightBrightness(float64(difference) / float64(wakeUpLength))
                        alarmInProgress = true
                    } else if alarmInProgress {
                        on = true
                        alarmInProgress = false
                    }
                } else {
                    setLightBrightness(0)
                }
            }
        } else {
            setLightBrightness(1)
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

    server = http.Server{Addr:"0.0.0.0:" + strconv.Itoa(Settings.Rest.Port), Handler: router}

    go func() {
        if err := server.ListenAndServe(); err != nil {
            fmt.Println(err)
        }
    }()
}

func initMQTT() {
    fmt.Println("MQTT starting")
    mqttAdaptor = mqtt.NewAdaptor(Settings.Mqtt.Broker, Settings.Mqtt.ClientID)
    mqttAdaptor.Connect()

    prefix := Settings.Mqtt.Prefix

    _, err := mqttAdaptor.OnWithQOS(prefix + "/on", 1, func(msg mqtt.Message) {
        if msg.Payload()[0] != 0 && msg.Payload()[0] != '0' {
            SetOn(true)
        } else {
            SetOn(false)
        }
        msg.Ack()
    })
    FatalErrorCheck(err)

    _, err = mqttAdaptor.OnWithQOS(prefix + "/alarm/+", 1, func(msg mqtt.Message) {
        topic := strings.Split(msg.Topic(), "/")
        day, _ := strconv.Atoi(topic[len(topic) - 2])
        SetAlarm(day, string(msg.Payload()))
    })
    FatalErrorCheck(err)

    _, err = mqttAdaptor.OnWithQOS(prefix + "/wake-up-length", 1, func(msg mqtt.Message) {
        SetWakeUpLength(string(msg.Payload()))
    })
    FatalErrorCheck(err)
    fmt.Println("MQTT started")
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

        SetAlarm(day, body.Time)
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
        SetOn(body.On)
    }
}

func SetWakeUpLength(input string) {
    wakeUpLength, _ = time.ParseDuration(input)
    fmt.Println("Wake up length set to " + wakeUpLength.String())
}

func SetAlarm(day int, input string) {
    alarm := strings.Split(input, ":")
    hour, _ := strconv.Atoi(alarm[0])
    minute, _ := strconv.Atoi(alarm[1])
    startTimes[day] = time.Duration(hour)  * time.Hour + time.Duration(minute) * time.Minute
    fmt.Println(day, " set to:", startTimes[day])
}

func SetOnPublish(on bool) {
    SetOn(on)
    mqttAdaptor.PublishWithQOS(
}

func SetOn(on bool) {
    fmt.Println("Light set to:", on)
    on = on
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
    if brightness != currentBrightness {
        currentBrightness = brightness
        fmt.Println("Brightness to ", brightness)
        var precision uint32 = 64
        cycle := uint32(onBrightness * brightness * float64(precision))
        if !Settings.Mock {
            light.DutyCycle(cycle, precision)
        }
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
