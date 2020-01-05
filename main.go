package main

import (
    "strings"
    "os"
    "time"
    "github.com/stianeikeland/go-rpio"
    "fmt"
    "bufio"
    "strconv"
    "math"
)

var startTimes = [7]time.Duration{-1, -1, -1, -1, -1, -1, -1}
var todayAlarm time.Time
var wakeUpLength time.Duration = time.Hour
var onBrightness float64 = 1
var startBrightness float64 = 0

var on bool = false
var alarmInProgress bool = false
var currentBrightness float64 = -1
var alarmCanceled = false

var light rpio.Pin
var button rpio.Pin
var zerocross rpio.Pin

var reader *bufio.Reader
var textWritten = false

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

    handleTimeTransitions()
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

func handleTimeTransitions() {
    clock := time.NewTicker(time.Second)
    i := 0
    for now := range(clock.C) {
        debug := i % 60 == 0
        if debug {
            fmt.Println("Time is ", now)
        }
        if alarmInProgress {
            difference := todayAlarm.Sub(now)
            if debug {
                fmt.Println(difference, " till alarm end")
            }
            if difference > 0 {
                setLightBrightness(math.Max((float64(wakeUpLength) - float64(difference)) / float64(wakeUpLength), 0))
            } else {
                alarmInProgress = false
                fmt.Println("Alarm finished")
                SetOnPublish(true)
            }
        } else {
            alarm := startTimes[now.Weekday()]
            if alarm >= 0 {
                checkTodayAlarm := getStartOfDay(now).Add(alarm)
                if todayAlarm != checkTodayAlarm && checkTodayAlarm.After(now) {
                    todayAlarm = checkTodayAlarm
                    fmt.Println("Alarm set for ",  todayAlarm)
                }
                if todayAlarm.After(now) {
                    tillAlarmStart := todayAlarm.Sub(now) - wakeUpLength
                    if !alarmCanceled && tillAlarmStart < 0 {
                        fmt.Println("Alarm starting")
                        alarmInProgress = true
                    } else if debug {
                        fmt.Println(tillAlarmStart, " till alarm start")
                    }
                }
            } else if debug {
                fmt.Println("No alarm for ", now.Weekday())
            }
        }
        i++
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
    if alarmInProgress {
        fmt.Println("Cancelling alarm")
        alarmCanceled = true
        alarmInProgress = false
        go func(){
            now := time.Now()
            alarm := startTimes[now.Weekday()]
            unsnoozedAlarm := getStartOfDay(now).Add(alarm)
            difference := unsnoozedAlarm.Sub(time.Now())
            time.Sleep(difference)
            alarmCanceled = false
            fmt.Println("Finished cancelled alarm")
        }()
    }
    if on {
        setLightBrightness(1)
    } else {
        setLightBrightness(0)
    }
}

func SetOnBrightness(brightness float64) {
    onBrightness = brightness
    fmt.Println("Brightness set to: ", brightness)
    if on {
        setLightBrightness(brightness)
    }
}

func SetStartBrightness(brightness float64) {
    startBrightness = brightness
    fmt.Println("Start brightness set to: ", startBrightness)
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
        if Settings.PullUp {
            button.Pull(rpio.PullUp)
        } else {
            button.Pull(rpio.PullDown)
        }

        if Settings.ZeroCrossPin >= 0 {
            zerocross = rpio.Pin(Settings.ZeroCrossPin)
            zerocross.Mode(rpio.Input)
            if Settings.PullUp {
                zerocross.Pull(rpio.PullUp)
            } else {
                zerocross.Pull(rpio.PullDown)
            }
            fmt.Println("Zero cross ", Settings.ZeroCrossPin)
        }
    } else {
        reader = bufio.NewReader(os.Stdin)
        go func(){
            for {
                read, _ := reader.ReadString('\n')
                if read == "a\n" {
                    textWritten = true
                } else {
                    textWritten = false
                }
            }
        }()
    }
    go processButtonPresses()
    setLightBrightness(0)
}

func processButtonPresses() {
    heldFor := 0
    ticker := time.NewTicker(time.Second / 60)
    for _ = range(ticker.C) {
        if (buttonPressed()) {
            heldFor++
        } else {
            if heldFor > 4 {
                if alarmInProgress {
                    if heldFor > 5 * 60 {
                        fmt.Println("Cancelling alarm...")
                        SetOnPublish(false)
                    } else {
                        todayAlarm = todayAlarm.Add(time.Minute * 5)
                        fmt.Println("Snoozing to ", todayAlarm)
                    }
                } else {
                    fmt.Println("Button pressed, toggling stat")
                    SetOnPublish(!on)
                }
            }
            heldFor = 0
        }
    }
    fmt.Println("proc done?")
    //fmt.Println("Zerocross read", zerocross.Read())
    //if zerocross.EdgeDetected() {
    //    fmt.Println("Zerocross edge", zerocross.EdgeDetected())
    //}
    //fmt.Println("button read", button.Read())
    //if button.EdgeDetected() {
    //    fmt.Println("button edge", button.EdgeDetected())
    //}
}

//Sets the brightness of the light with 1 being full on
//and 0 being off.
func setLightBrightness(brightness float64) {
    if brightness != currentBrightness {
        currentBrightness = brightness
        fmt.Println("Brightness to ", brightness)
        if brightness > 0 && brightness < startBrightness {
            brightness = startBrightness
        }
        var precision uint32 = 120
        cycle := uint32(onBrightness * brightness * float64(precision))
        if !Settings.Mock {
            fmt.Println("Duty to ", cycle , " / ", precision)
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
        return (button.Read() == rpio.High)  == !Settings.PullUp
    } else {
        return textWritten
    }
}
