package main

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "net/http"
    "strconv"
    "gobot.io/x/gobot/platforms/mqtt"
    "strings"
)


var mqttAdaptor *mqtt.Adaptor
var server http.Server

func initApi() {
    if Settings.Rest.Enabled {
        initServer()
    }
    if Settings.Mqtt.Enabled {
        initMQTT()
    }
}

func closeApi() {
    if Settings.Rest.Enabled {
        closeServer()
    }
    if Settings.Mqtt.Enabled {
        mqttAdaptor.Disconnect()
    }
}

func initServer() {
    router := mux.NewRouter()
    router.HandleFunc("/alarm/{day:[0-6]}", dayAlarmHandler)
    router.HandleFunc("/light", dayAlarmHandler)

    server = http.Server{Addr:"0.0.0.0:" + strconv.Itoa(Settings.Rest.Port), Handler: router}

    go func() {
        if err := server.ListenAndServe(); err != nil {
            FatalErrorCheck(err)
        }
    }()
}

func initMQTT() {
    Info.Println("MQTT starting")
    mqttAdaptor = mqtt.NewAdaptor(Settings.Mqtt.Broker, Settings.Mqtt.ClientID)
    err := mqttAdaptor.Connect()
    FatalErrorCheck(err)

    prefix := Settings.Mqtt.Prefix

    _, err = mqttAdaptor.OnWithQOS(prefix + "/on", 1, func(msg mqtt.Message) {
	defer dontPanic()
        if msg.Payload()[0] != 0 && msg.Payload()[0] != '0' {
            SetOn(true)
        } else {
            SetOn(false)
        }
        msg.Ack()
    })
    FatalErrorCheck(err)

    _, err = mqttAdaptor.OnWithQOS(prefix + "/alarm/+", 1, func(msg mqtt.Message) {
	defer dontPanic()
        topic := strings.Split(msg.Topic(), "/")
        day, err := strconv.Atoi(topic[len(topic) - 1])
        if ErrorCheck(err) {
            return
        }
        SetAlarm(day, string(msg.Payload()))
    })
    FatalErrorCheck(err)

    _, err = mqttAdaptor.OnWithQOS(prefix + "/wake-up-length", 1, func(msg mqtt.Message) {
	defer dontPanic()
        SetWakeUpLength(string(msg.Payload()))
    })
    FatalErrorCheck(err)

    _, err = mqttAdaptor.OnWithQOS(prefix + "/brightness", 1, func(msg mqtt.Message) {
	defer dontPanic()
			bright, err := strconv.ParseFloat(string(msg.Payload()), 64)
		ErrorCheck(err)
		SetOnBrightness(bright)
    })
    FatalErrorCheck(err)

    Info.Println("MQTT started")
}

func dayAlarmHandler(response http.ResponseWriter, request *http.Request){
	defer dontPanic()
    if request.Method == "PUT" {
        day, err := strconv.Atoi(mux.Vars(request)["day"])
        var body struct {
            Time string
        }

        err = json.NewDecoder(request.Body).Decode(&body)
        if err != nil {
            http.Error(response, err.Error(), http.StatusBadRequest)
            Error.Println(err)
            return
        }

        SetAlarm(day, body.Time)
    }
}

func LightHandler(response http.ResponseWriter, request *http.Request) {
	defer dontPanic()
    if request.Method == "PUT" {
        var body struct {
            On bool
        }

        err := json.NewDecoder(request.Body).Decode(&body)
        if err != nil {
            http.Error(response, err.Error(), http.StatusBadRequest)
            Error.Println(err)
            return
        }
        SetOn(body.On)
    }
}

func SetOnPublish(on bool) {
    SetOn(on)
    var toPub []byte
    if on {
        toPub = []byte{'1'}
    } else {
        toPub = []byte{'0'}
    }
    mqttAdaptor.PublishWithQOS(Settings.Mqtt.Prefix + "/on", 1, toPub)
}

func dontPanic(location string) {
    r := recover()
    if r != nil {
        Error.Println(location, r)
    }
}

func closeServer() {
    server.Close()
}

func closeMQTT() {
    mqttAdaptor.Disconnect()
}
