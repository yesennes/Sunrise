package main

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "fmt"
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
        defer closeServer()
    }
    if Settings.Mqtt.Enabled {
        initMQTT()
        defer mqttAdaptor.Disconnect()
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

func dayAlarmHandler(response http.ResponseWriter, request *http.Request){
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

func SetOnPublish(on bool) {
    SetOn(on)
    mqttAdaptor.PublishWithQOS(Settings.Mqtt.Prefix + "/on", 1, []byte{'0'})
}

func closeServer() {
    server.Close()
}

func closeMQTT() {
    mqttAdaptor.Disconnect()
}
