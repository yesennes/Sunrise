package main

import (
    "strings"
    "os"
    "time"
    "github.com/stianeikeland/go-rpio"
    "fmt"
    "bufio"
    "strconv"
)

var startTimes = [7]time.Duration{-1, -1, -1, -1, -1, -1, -1}
var todayAlarm time.Time
var wakeUpLength time.Duration = time.Hour
var onBrightness float64 = .25
var minBrightness float64 = .1

var on bool = false
var alarmInProgress bool = false
var currentBrightness float64 = -1

var light rpio.Pin
var button rpio.Pin

func main() {
    fmt.Println("Starting Sunrise")
    if len(os.Args) > 1 {
        fmt.Println("Loading from " + os.Args[1])
        LoadConfig(os.Args[1])
    } else {
        fmt.Println("Loading from /etc/Sunrise.yaml")
        LoadConfig("/etc/Sunrise.yaml")
    }

    defer closeApi()

    initHardware()
    fmt.Println("Hardware initialized")

    initApi()
    fmt.Println("Api started")

    waitForAlarms()
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
            if alarm >= 0 {
                if !alarmInProgress {
                    todayAlarm = getStartOfDay(now).Add(alarm)
                    alarmInProgress = true
                }
                if buttonPressed() || checkButtonFallingEdge() {
                    todayAlarm = todayAlarm.Add(time.Minute)
                }
                difference := now.Sub(todayAlarm)
                if difference > 0 {
                    if difference < wakeUpLength {
                        setLightBrightness(float64(difference) / float64(wakeUpLength))
                    } else if alarmInProgress {
                        SetOnPublish(true)
                        alarmInProgress = false
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


func SetOn(newState bool) {
    fmt.Println("Light set to:", newState)
    on = newState
    if !alarmInProgress {
        if on {
            setLightBrightness(1)
        } else {
            setLightBrightness(0)
        }
    }
}

func initHardware() {
    if !Settings.Mock {
        err := rpio.Open()
        FatalErrorCheck(err)
        light = rpio.Pin(Settings.LightPin)
        light.Mode(rpio.Pwm)
        light.Freq(76000)

        button = rpio.Pin(Settings.ButtonPin)
        button.Mode(rpio.Input)
        button.Pull(rpio.PullDown)
        button.Detect(rpio.FallEdge)
    }
    go func() {
        ticker := time.NewTicker(time.Second / 60)
        for _ = range(ticker.C) {
            if (!alarmInProgress && checkButtonFallingEdge()) {
                fmt.Println("Button pressed")
                SetOnPublish(!on)
            }
        }
    }()
    setLightBrightness(0)
}

//Sets the brightness of the light with 1 being full on
//and 0 being off.
func setLightBrightness(brightness float64) {
    if brightness != currentBrightness {
        currentBrightness = brightness
        fmt.Println("Brightness to ", brightness)
        if brightness > 0 || brightness < minBrightness {
            brightness = minBrightness
        }
        var precision uint32 = 120
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

func buttonPressed() bool {
    if !Settings.Mock {
        return button.Read() == rpio.High
    }
    return false
}

func checkButtonFallingEdge() bool {
    if !Settings.Mock {
        edge := button.EdgeDetected()
        return edge
    }
    return false
}
