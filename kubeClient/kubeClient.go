// Package kubeClient provide the functionality to initalize a MQTT connection and update the device twin
// in a feauture version this package should also provide to handle incomming message via a callback function
package kubeClient

import (
	"log"
	"crypto/tls"
	"sync"
	"time"
	"encoding/json"

	MQTT "github.com/eclipse/paho.mqtt.golang"
)

type Token interface {
	Wait() bool
	WaitTimeout(time.Duration) bool
	Error() error
}

//DeviceStateUpdate is the structure used in updating the device state
type DeviceStateUpdate struct {
	State string `json:"state,omitempty"`
}

//BaseMessage the base struct of event message
type BaseMessage struct {
	EventID   string `json:"event_id"`
	Timestamp int64  `json:"timestamp"`
}

//TwinValue the struct of twin value
type TwinValue struct {
	Value    *string	`json:"value, omitempty"`
	Metadata *ValueMetadata `json:"metadata,omitempty"`
}

//ValueMetadata the meta of value
type ValueMetadata struct {
	Timestamp int64 `json:"timestamp, omitempty"`
}

//TypeMetadata the meta of value type
type TypeMetadata struct {
	Type string `json:"type,omitempty"`
}

//TwinVersion twin version
type TwinVersion struct {
	CloudVersion int64 `json:"cloud"`
	EdgeVersion  int64 `json:"edge"`
}

// MsgTwin the structe of device twin
type MsgTwin struct {
	Actual		*TwinValue	`json:"temperature,omitempty"`
	Optional	*bool		`json:"optional,omitempty"`
	Metadata	*TypeMetadata	`json:"metadata,omitempty"`
	ExpectedVersion	*TwinValue	`json:"expected_version,omitempty"`
	ActualVersion	*TwinVersion	`json:"actual_version,omitempty"`
}

// DeviceTwinUdpdate the struct of device twin update
type DeviceTwinUpdate struct {
	BaseMessage
	Twin map[string]*MsgTwin	`json:twin"`
}

var (
	Prefix = "$hw/events/device"
	StateUpdateSuffix = "/state/update"
	TwinUpdateSuffix = "/twin/update"
	TwinCloudUpdateSuffix = "/twin/cloud_update"
	/*
	 * not needed beacause this programm will not receive any information from the edge / cloud
	 * TwinGetResultSuffix = "/twin/get/result"
 	 * TwinGetSuffix = "/twin/get"
	*/
)

var token_client Token
var clientOpts *MQTT.ClientOptions
var client MQTT.Client
var wg sync.WaitGroup
var deviceID string

// mqttConfig crate the mqtt client config
func mqttConfig(server, clientID, user, password string) *MQTT.ClientOptions {
	options := MQTT.NewClientOptions().AddBroker(server).SetClientID(clientID).SetCleanSession(true)
	if user != "" {
		options.SetUsername(user)
		if password != "" {
			options.SetPassword(password)
		}
	}
	tlsConfig := &tls.Config{InsecureSkipVerify: true, ClientAuth: tls.NoClientCert};
	options.SetTLSConfig(tlsConfig)
	return options
}

// changeSensorStatus function is used to change the state of this sensor
func changeSensorStatus(state string) {
	log.Println("Changing the state of the device to ", state)
	var sensorStateUpdate DeviceStateUpdate
	sensorStateUpdate.State = state
	messageBody, err := json.Marshal(sensorStateUpdate)
	if err != nil {
		log.Panicln(err)
	}
	statusUpdate := Prefix + deviceID + StateUpdateSuffix
	token_client = client.Publish(statusUpdate, 0, false, messageBody)
	if token_client.Wait() && token_client.Error() != nil {
		log.Panicln("client.publish() Error in sensor state update is: ", token_client.Error())
	}
}

// changeTwinValue sends the updated twin value to the edge through the MQTT broker
func changeTwinValue(updateMessage DeviceTwinUpdate){
	messageBody, err := json.Marshal(updateMessage)
	if err != nil {
		log.Println("Error: ", err)
	}
	topic := Prefix + deviceID + TwinUpdateSuffix
	token_client = client.Publish(topic, 0, false, messageBody)
	if token_client.Wait() && token_client.Error() != nil {
		log.Println("client.publish() Error in device twin update is: ", token_client.Error())
	}
}

// syncToCloud function syncs the updated device twin infromation to the cloud
func syncToCloud(message DeviceTwinUpdate) {
	topic := Prefix + deviceID + TwinCloudUpdateSuffix
	messageBody, err := json.Marshal(message)
	if err != nil {
		log.Println("syncToCoud marshal error is: ", err)
	}
	token_client = client.Publish(topic, 0, false, messageBody)
	if token_client.Wait() && token_client.Error() != nil {
		log.Println("client.publish() erro in device twin update to cloud is: ", token_client.Error())
	}
}

// OnSubMessageReceived callback function wich is called when message is received
//func onSubMessageReceived(client MQTT.Client, message MQTT.Message) {
//	err := json.Unmarshal(message.Payload(), &deviceTwinResult)
//	if err != nil {
//		log.Println("Error in unmarshalling: ", err)
//	}
//}

// createActualUpdateMessage function is used to create the device twin update message
func createActualUpdateMessage(actualValue string) DeviceTwinUpdate {
	var message DeviceTwinUpdate
	actualMap := map[string]*MsgTwin{"CPU_Temperatur": {Actual: &TwinValue{Value: &actualValue}, Metadata: &TypeMetadata{Type: "Updated"}}}
	message.Twin = actualMap
	return message
}

// getTwin function is used to get the device twin detials from the edge
//func getTwin(message DeviceTwinUpdate){
//	topic := Prefix + deviceID + TwinETGetSuffix
//	messageBody, err := json.Marshal(message)
//	if err != nil {
//		log.Println("Error at marshal: ", err)
//	}
//	token_client = client.Publish(topic, 0, false, messageBody)
//	if token_client.Wait() && token_client.Error() != nil {
//		fmt.Println("client.publsih() Error in get device twin is: ", token_client.Error())
//	}
//}

// subscribe function subscribes the device twin information through the MQTT broker
//func subscribe() {
//	for {
//		topic := Prefix + deviceID + TwinETGetResultSuffix
//		token_client = client.Subscibe(topic, 0, OnSubMessageReceived)
//		if token_client.Wait() && token_client.Erro() != nil {
//			log.Println("subscribe() Error in device twin result get is: ", token_client.Error())
//		}
//		time.Sleep(1 * time.Second)
//		if deviceTwinResult.Twin != nil {
//			wg.Done()
//			break
//		}
//	}
//}

// TODO write a function which takes a callback method which will be executed when a message has arrived over MQTT

// Update this function is used to update values on the edge and in the cloud
func Update(value string) {
	log.Println("Syncing to edge")
	updateMessage := createActualUpdateMessage(value)
	changeTwinValue(updateMessage)
	time.Sleep(2 * time.Second)
	log.Println("Syncing to cloud")
	syncToCloud(updateMessage)
}

// Init hit initalise the MQTT connection and set the used ipAddress and deviceID
// in a feature version this function also register the callback method to handle incomming messages
// ipAddress and deviceID has to be set! If you don't want to add an user or an password in the 
// MQTT Connection set user and password to nil
func Init(ipAddress, id, user, password string) {
	deviceID = id
	clientOpts = mqttConfig(ipAddress, deviceID, user, password)
	client = MQTT.NewClient(clientOpts)
	if token_client = client.Connect(); token_client.Wait() && token_client.Error() != nil {
		log.Println("client.Connect() Error is: ", token_client.Error())
	}
}