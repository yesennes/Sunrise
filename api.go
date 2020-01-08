package main

import (
    "strings"
    "time"
    "encoding/json"
    "github.com/gorilla/mux"
    "net/http"
    "strconv"
    "github.com/eclipse/paho.mqtt.golang"
)


var mqttClient mqtt.Client
var server http.Server

var prefix string

func initApi() {
    if Settings.Rest.Enabled {
        initServer()
    }
    if Settings.Mqtt.Enabled {
        initMQTT()
    }
}

func closeApi() {
    Info.Println("Shutting down api")
    if Settings.Rest.Enabled {
        closeServer()
    }
    if Settings.Mqtt.Enabled {
        publish("/$state", "disconnected")
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
    prefix = "homie/" + Settings.Mqtt.DeviceID
    Info.Println("MQTT starting")
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

    subscribe("/light/on", func(client mqtt.Client, msg mqtt.Message) {
        on, _ := strconv.ParseBool(string(msg.Payload()))
        SetOn(on)
        msg.Ack()
    })

    publish("/light/brightness/$name", "Brightness")
    publish("/light/brightness/$datatype", "float")
    publish("/light/brightness/$settable", "true")
    publish("/light/brightness/$format", "0:1")
    publish("/light/brightness/$unit", "%")

    subscribe("/light/brightness", func(client mqtt.Client, msg mqtt.Message) {
        bright, err := strconv.ParseFloat(string(msg.Payload()), 64)
		ErrorCheck(err)
		SetOnBrightnessPublish(bright)
    })

    publish("/alarm/$name", "Alarm")
    publish("/alarm/$type", "")

    allDays := make([]string, 7)
    for i := time.Sunday; i <= time.Saturday; i++ {
        str := i.String()

        allDays[i] = str

        publish("/alarm/" + str + "/$name", "Alarm for " + str)
        publish("/light/" + str + "/$datatype", "string")
        publish("/light/" + str + "/$settable", "true")
        publish("/light/" + str + "/$unit", "time of day")

        subscribe("/alarm/" + i.String(), func(client mqtt.Client, msg mqtt.Message) {
            SetAlarm(i, string(msg.Payload()))
        })
    }

    publish("/alarm/$properties", strings.Join(allDays, ",") + ",wake-up-length")

    publish("/light/wake-up-length/$name", "Alarm Dimming Time")
    publish("/light/wake-up-length/$datatype", "integer")
    publish("/light/wake-up-length/$settable", "true")
    publish("/light/wake-up-length/$unit", "minutes")
    subscribe("/wake-up-length", func(client mqtt.Client, msg mqtt.Message) {
        SetWakeUpLength(string(msg.Payload()))
    })

    publish("/$state", "ready")
    Info.Println("MQTT started")
}


func publish(topic string, payload interface{}){
    checkMQTTError(mqttClient.Publish(prefix + topic, 1, true, payload))
}

func subscribe(topic string, handler mqtt.MessageHandler) {
    checkMQTTError(mqttClient.Subscribe(prefix + topic + "/set", 1, func(client mqtt.Client, msg mqtt.Message) {
        handler(client, msg)
        publish(topic, msg.Payload())
        msg.Ack()
    }))

    checkMQTTError(mqttClient.Subscribe(prefix + topic, 1, func(client mqtt.Client, msg mqtt.Message) {
        handler(client, msg)
        checkMQTTError(mqttClient.Unsubscribe(prefix + topic))
        msg.Ack()
    }))
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

        SetAlarm(time.Weekday(day), body.Time)
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

func checkMQTTError(token mqtt.Token) mqtt.Token {
    go func(){
        token.Wait()
        ErrorCheck(token.Error())
    }()
    return token
}

func closeServer() {
    server.Close()
}
