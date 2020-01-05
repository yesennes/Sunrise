package main

import (
    "encoding/json"
    "github.com/gorilla/mux"
    "net/http"
    "strconv"
    "github.com/eclipse/paho.mqtt.golang"
    "strings"
)


var mqttClient mqtt.Client
var server http.Server

var prefix = "homie/" + Settings.Mqtt.DeviceID

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
        mqttClient.Disconnect(50)
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
    prefix := "homie/" + Settings.Mqtt.DeviceID

    options := mqtt.NewClientOptions()
    options.AddBroker(Settings.Mqtt.Broker)
    options.SetClientID(Settings.Mqtt.ClientID)
    options.SetConnectionLostHandler(func(client mqtt.Client, err error) {
        ErrorCheck(err)
    })
    options.SetOnConnectHandler(initMQTTTopics)
    options.SetWill(prefix + "/$state", "lost", 1, true)

    mqttClient = mqtt.NewClient(options)

    token := mqttClient.Connect()
    token.Wait()
    FatalErrorCheck(token.Error())
}

func initMQTTTopics(client mqtt.Client) {
    publish("/$state", "init")
    publish("/$homie", "4.0.0")
    publish("/$name", "Sunrise Alarm Clock")
    publish("/$nodes", "light,alarm")

    publish("/light/$name", "Light")
    publish("/light/$type", "")
    publish("/light/$properties", "on,brightness")

    publish("/light/on/$name", "On/off state")
    publish("/light/on/$datatype", "boolean")
    publish("/light/on/$settable", "true")

    subscribe("/light/on/set", func(client mqtt.Client, msg mqtt.Message) {
        if msg.Payload()[0] != 0 && msg.Payload()[0] != '0' {
            SetOn(true)
        } else {
            SetOn(false)
        }
        msg.Ack()
    })

    publish("/light/on/$name", "Brightness")
    publish("/light/on/$datatype", "float")
    publish("/light/on/$settable", "true")
    publish("/light/on/$format", "0:1")

    subscribe("/light/brightness/set", func(client mqtt.Client, msg mqtt.Message) {
			bright, err := strconv.ParseFloat(string(msg.Payload()), 64)
		ErrorCheck(err)
		SetOnBrightnessPublish(bright)
        msg.Ack()
    })

    subscribe("/alarm/+", func(client mqtt.Client, msg mqtt.Message) {
        topic := strings.Split(msg.Topic(), "/")
        day, err := strconv.Atoi(topic[len(topic) - 1])
        if ErrorCheck(err) {
            return
        }
        SetAlarm(day, string(msg.Payload()))
        msg.Ack()
    })

    subscribe("/wake-length", func(client mqtt.Client, msg mqtt.Message) {
        SetWakeUpLength(string(msg.Payload()))
        msg.Ack()
    })

    Info.Println("MQTT started")
}


func publish(topic string, payload interface{}){
    checkMQTTError(mqttClient.Publish(prefix + topic, 1, true, payload))
}

func subscribe(topic string, handler mqtt.MessageHandler) {
    checkMQTTError(mqttClient.Subscribe(prefix + topic, 1, handler))
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
            Error.Println(err)
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
    checkMQTTError(mqttClient.Publish("homie/" + Settings.Mqtt.DeviceID + "/light/on", 1, true, toPub))
}

func SetOnBrightnessPublish(brightness float64) {
    SetOnBrightness(brightness)
    publish("/light/brightness", strconv.FormatFloat(brightness, 'f', -1, 64))
}

func checkMQTTError(token mqtt.Token) {
    go func(){
        token.Wait()
        ErrorCheck(token.Error())
    }()
}

func closeServer() {
    server.Close()
}
