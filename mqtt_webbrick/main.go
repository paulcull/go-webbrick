package main

import (
	"encoding/json" // json encoding
	"fmt"           // For outputting messages
	//"os"        // For OS Interaction
	//"os/signal" // For picking up the signal

	"github.com/paulcull/go-webbrick" // For controlling Orvibo stuff

	//"github.com/davecgh/go-spew/spew" // For neatly outputting stuff
	"strconv"
	"time" // For setInterval()

	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
)

// No config provided? Set up some defaults
func defaultConfig() *webbrick.WebbrickDriverConfig {

	// Set the default Configuration
	//log = logger.GetLogger(info.Name)
	return &webbrick.WebbrickDriverConfig{
		Name:            "PKHome",
		Initialised:     false,
		NumberOfDevices: 0,
		PollingMinutes:  5,
		PollingActive:   false,
	}
}

func main() {

	// Set up channel on which to send signal notifications.
	//sigc := make(chan os.Signal, 1)
	//signal.Notify(sigc, os.Interrupt, os.Kill)

	// Create an MQTT Subscribe Client.
	fmt.Println("Setting up mqtt client...")
	s_cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})

	// Terminate the Client.
	//defer cli.Terminate()

	// Connect to the MQTT Server.
	fmt.Println("Setting up mqtt client...Connecting...")
	err := s_cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  "auto-openhab-n1.lan:1883",
		ClientID: []byte("openHAB"),
	})
	if err != nil {
		fmt.Println(err)
		panic(err)
	} else {
		fmt.Println("Setting up mqtt client...Connecting...done")
	}

	// Subscribe to topics.
	fmt.Println("Setting up mqtt client...Subscribing...")

	err = s_cli.Subscribe(&client.SubscribeOptions{
		SubReqs: []*client.SubReq{
			&client.SubReq{
				TopicFilter: []byte("#"),
				QoS:         mqtt.QoS0,
				// Define the processing of the message handler.
				Handler: func(topicName, message []byte) {
					fmt.Println(string(topicName), string(message))
				},
			},
			// &client.SubReq{
			// 	TopicFilter: []byte("bar/#"),
			// 	QoS:         mqtt.QoS1,
			// 	Handler: func(topicName, message []byte) {
			// 		fmt.Println(string(topicName), string(message))
			// 	},
			// },
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Setting up mqtt client...Subscribing...done")

	// Unsubscribe from topics.
	// err = cli.Unsubscribe(&client.UnsubscribeOptions{
	// 	TopicFilters: [][]byte{
	// 		[]byte("foo"),
	// 	},
	// })
	// if err != nil {
	// 	panic(err)
	// }

	// Wait for receiving a signal.
	//<-sigc

	//Disconnect the Network Connection.
	fmt.Println("******** Setting up mqtt client...Dis-connecting...")
	if err := s_cli.Disconnect(); err != nil {
		panic(err)
	}
	fmt.Println("Setting up mqtt client...Dis-connecting...done")

	ready, err := webbrick.Prepare(defaultConfig()) // You ready?
	if ready == true {                              // Yep! Let's do this!
		for { // Loop forever
			select { // This lets us do non-blocking channel reads. If we have a message, process it. If not, check for UDP data and loop
			case msg := <-webbrick.Events:
				fmt.Println(" **** Event for ", msg.Name, "received from... ", msg.DeviceInfo.IP.String())
				//strMsg := fmt.Sprintf("%+v", msg)
				strMsgJSON, _ := json.Marshal(msg)
				strMsg := string(strMsgJSON)
				fmt.Println(strMsg)
				//sent, err := publishMessage(cli, "webbrick/"+strconv.Itoa(msg.DeviceInfo.ID)+"/"+msg.Name+"/"+msg.DeviceInfo.DevID, strMsg)
				sent, err := publishMessage(strMsg, "webbrick/"+strconv.Itoa(msg.DeviceInfo.ID)+"/"+msg.Name+"/"+msg.DeviceInfo.DevID)
				if err != nil && sent == false {
					fmt.Println(" !!!!!!!!!!!!!!!!! Error in publishMessage")
					//panic(err)
				}
				//fmt.Println(sent)

			default:
				webbrick.CheckForMessages()
			}

		}
		//fmt.Println(" **** List of Devices ****")
		//webbrick.ListDevices()
	} else {
		fmt.Println("Error:", err)
	}

	// Wait for receiving a signal.
	//<-sigc

	// // Disconnect the Network Connection.
	// if err := cli.Disconnect(); err != nil {
	// 	panic(err)
	// }

}

//func publishMessage(cli *client.Client, message string, topic string) (bool, error) {
func publishMessage(message string, topic string) (bool, error) {

	// Create an MQTT Client.
	fmt.Println("Setting up mqtt client publish...")
	p_cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})

	// Terminate the Client.
	defer p_cli.Terminate()

	// Connect to the MQTT Server.
	fmt.Println("Setting up mqtt client publish...Connecting...")
	err := p_cli.Connect(&client.ConnectOptions{
		Network:  "tcp",
		Address:  "auto-openhab-n1.lan:1883",
		ClientID: []byte("openHAB"),
	})
	if err != nil {
		fmt.Println(err)
		return false, err
		//panic(err)
	} else {
		fmt.Println("Setting up mqtt client publish...Connecting...done")
	}

	// check that we have the topic set
	if topic == "" {
		topic = "webbrick"
	}
	//publish
	fmt.Println("Setting up mqtt client publish...Publishing...")
	err = p_cli.Publish(&client.PublishOptions{
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
		//panic(err)
	}
	fmt.Println("Setting up mqtt client publish...Publishing...done")

	fmt.Println("Setting up mqtt client publish...Dis-connecting...")
	if err := p_cli.Disconnect(); err != nil {
		panic(err)
		return false, err
	}
	fmt.Println("Setting up mqtt client publish...Dis-connecting...done")

	//true status
	return true, nil
}

func setInterval(what func(), delay time.Duration) chan bool {
	stop := make(chan bool)

	go func() {
		for {
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
