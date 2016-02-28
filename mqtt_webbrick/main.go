package main

import (
	"encoding/json"                     // json encoding
	"fmt"                               // For outputting messages
	"github.com/paulcull/go-webbrick"   // For controlling Orvibo stuff
	"github.com/yosssi/gmq/mqtt"        //mqtt libraries
	"github.com/yosssi/gmq/mqtt/client" //mqtt libraries
	"os"                                // For OS Interaction
	"os/signal"                         // For picking up the signal
	"strconv"
	"strings"
	"syscall" // Pick up for when running as systemctl service
	"time"
)

// No config provided? Set up some defaults
func defaultConfig() *webbrick.WebbrickDriverConfig {

	// Set the default Configuration
	return &webbrick.WebbrickDriverConfig{
		Name:            "PKHome",
		Initialised:     false,
		NumberOfDevices: 0,
		PollingMinutes:  5,
		PollingActive:   false,
	}
}

func mqttConfig() *client.ConnectOptions {
	return &client.ConnectOptions{
		Network:  "tcp",
		Address:  "localhost:1883",
		ClientID: []byte("webbrickBridge"),
	}
}

////////////////////
// main proc
////////////////////
// 1. sets up a catch for interrupts
// 2. connects to mqtt broker
// 3. creates a heartbeat provider
// 4. set the redirect for interrupts
// 5. subscribe to the inbound message channels
// 6. connect to webbrick library and listen for events
////////////////////
func main() {

	////////////////////
	// Set up channel on which to send signal notifications.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill,
		syscall.SIGTERM, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT)

	////////////////////
	//Create an MQTT Subscribe Client.
	fmt.Println("Setting up mqtt client...")
	cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})

	////////////////////
	// Connect to the MQTT Server.
	fmt.Println("Setting up mqtt client...Connecting...")
	err := cli.Connect(mqttConfig())
	if err != nil {
		fmt.Println(err)
		panic(err)
	} else {
		fmt.Println("Setting up mqtt client...Connecting...done")
	}

	////////////////////
	// setup a hearbeat service publisher
	ping := func() {
		sent, err := publishMessage(cli, "Alive", "webbrick/bridge/heartbeat")
		if err != nil && sent == false {
			fmt.Println(" !!!!!!!!!!!!!!!!! Error in publishMessage")
			panic(err)
		}
	}
	heartbeat := setInterval(ping, 59*time.Second)

	////////////////////
	// catch the exit signal and tidy up connections cleanly
	go func() {
		<-sigc
		cleanup(cli, heartbeat)
		os.Exit(1)
	}()

	////////////////////
	// Subscribe to topics.
	fmt.Println("Setting up mqtt client...Subscribing...")
	err = cli.Subscribe(&client.SubscribeOptions{
		SubReqs: []*client.SubReq{
			&client.SubReq{
				TopicFilter: []byte("webbrick/to/#"),
				QoS:         mqtt.QoS1,
				Handler: func(topicName, message []byte) {
					// fmt.Println(string(topicName), string(message))
					actOnMessage(topicName, message)
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Setting up mqtt client...Subscribing...done")

	////////////////////
	// connect to webbrick library
	ready, err := webbrick.Prepare(defaultConfig()) // You ready?
	if ready == true {                              // Yep! Let's do this!
		for { // Loop forever
			fmt.Println((" *** In the loop waiting for UDP messages..."))
			select { // This lets us do non-blocking channel reads. If we have a message, process it. If not, check for UDP data and loop
			case msg := <-webbrick.Events:
				fmt.Println(" **** Event for ", msg.Name, "received from... ", msg.DeviceInfo.IP.String())
				strMsgJSON, _ := json.Marshal(msg)
				strMsg := string(strMsgJSON)
				_msg := ""
				fmt.Println(strMsg)
				if msg.DeviceInfo.Level > 0 {
					_msg = strconv.FormatFloat(msg.DeviceInfo.Level, 'G', -1, 32)
				} else {
					_msg = strconv.FormatBool(msg.DeviceInfo.State)
				}
				//sent, err := publishMessage(cli, strMsg, "webbrick/from/"+
				sent, err := publishMessage(cli, _msg, "webbrick/from/"+
					strconv.Itoa(msg.DeviceInfo.BrickID)+ //Brick Node
					"/"+
					strconv.Itoa(msg.DeviceInfo.Type)+ // Type ID
					"/"+
					strconv.Itoa(msg.DeviceInfo.Channel)+ // type channel
					"/"+
					msg.DeviceInfo.DevID)
				if err != nil && sent == false {
					fmt.Println(" !!!!!!!!!!!!!!!!! Error in publishMessage")
					panic(err)
				}
				fmt.Println(sent)
				if msg.Name == "newwebbrickfound" { // if its a new webbrick - then go and get all the details
					webbrick.PollWBStatus(msg.DeviceInfo.DevID)
				}
			default:
				//fmt.Println(" **** Checking for messages ****")
				webbrick.CheckForMessages()
			}

		}
		fmt.Println(" **** List of Devices ****")
		webbrick.ListDevices()
	} else {
		fmt.Println("Error:", err)
	}

	// Wait for receiving a signal.
	//<-sigc
}

func actOnMessage(topicName, message []byte) {

	fmt.Println(" **************************** in actOnMessage ***")
	fmt.Println(string(topicName))
	inboundDev := string(topicName)[strings.LastIndex(string(topicName), "/")+1 : len(string(topicName))]
	fmt.Println(inboundDev)
	fmt.Println(string(message))
	fmt.Println(" **************************** in actOnMessage ***")

}

func publishMessage(cli *client.Client, message string, topic string) (bool, error) {
	fmt.Println("Setting up mqtt client publish...Publishing...", topic, message)
	//err := p_cli.Publish(&client.PublishOptions{
	err := cli.Publish(&client.PublishOptions{
		QoS:       mqtt.QoS0,
		TopicName: []byte(topic),
		Message:   []byte(message),
	})
	//check for err
	if err != nil {
		fmt.Println(" **** error in trying to publish ****")
		fmt.Println(topic)
		fmt.Println(message)
		fmt.Println(" **** error in trying to publish ****")
		return false, err
		panic(err)
	}
	fmt.Println("Setting up mqtt client publish...Publishing...done")
	//true status
	return true, nil
}

func cleanup(cli *client.Client, heartbeat chan bool) {

	////////////////////
	// stop heartbear
	heartbeat <- false

	webbrick.ListDevices()

	////////////////////
	// unsubscribe topics
	fmt.Println("******** Setting up mqtt client...Unsubscribing...")
	err := cli.Unsubscribe(&client.UnsubscribeOptions{
		TopicFilters: [][]byte{
			[]byte("webbrick/to/#"),
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("******** Setting up mqtt client...Unsubscribing...done")

	////////////////////
	// Disconnect the Network Connection.
	fmt.Println("******** Setting up mqtt client...Dis-connecting...")
	if err := cli.Disconnect(); err != nil {
		panic(err)
	}
	fmt.Println("******** Setting up mqtt client...Dis-connecting...done")

	////////////////////
	// Terminate theclient.
	fmt.Println("******** Setting up mqtt client...Terminating...")
	cli.Terminate()
	fmt.Println("******** Setting up mqtt client...Terminating...done")

}

func setInterval(what func(), delay time.Duration) chan bool {
	stop := make(chan bool)

	go func() {
		for {
			//fmt.Println("Running Set Interval ****************")
			what()
			select {
			case <-time.After(delay):
			case <-stop:
				return
			}
		}
	}()

	return stop
}
