package main

import (
	"fmt" // For outputting messages
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

	// Create an MQTT Client.
	cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})

	// Terminate the Client.
	defer cli.Terminate()

	// Connect to the MQTT Server.
	err := cli.Connect(&client.ConnectOptions{
		Network: "tcp",
		Address: "auto-openhab-n1.lan:1883",
		//Address:  "iot.eclipse.org:1883",
		ClientID: []byte("openHAB"),
		//ClientID: []byte("example-client"),
	})
	if err != nil {
		panic(err)
	}

	// Subscribe to topics.
	err = cli.Subscribe(&client.SubscribeOptions{
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

	// Publish a message.
	// err = cli.Publish(&client.PublishOptions{
	// 	QoS:       mqtt.QoS0,
	// 	TopicName: []byte("bar/baz"),
	// 	Message:   []byte("testMessage"),
	// })
	// if err != nil {
	// 	panic(err)
	// }

	// Unsubscribe from topics.
	err = cli.Unsubscribe(&client.UnsubscribeOptions{
		TopicFilters: [][]byte{
			[]byte("foo"),
		},
	})
	if err != nil {
		panic(err)
	}

	ready, err := webbrick.Prepare(defaultConfig()) // You ready?
	if ready == true {                              // Yep! Let's do this!
		for { // Loop forever
			select { // This lets us do non-blocking channel reads. If we have a message, process it. If not, check for UDP data and loop
			case msg := <-webbrick.Events:
				fmt.Println(" **** Event for ", msg.Name, "received...")
				strMsg := fmt.Sprintf("%#v", msg)
				fmt.Println(strMsg)
				sent, err := publishMessage(cli, msg.Name+"/"+msg.DeviceInfo.DevID, strMsg)
				if err != nil {
					fmt.Println("Error in publishMessage")
					panic(err)
				}
				fmt.Println(sent)
				//publishMessage(cli, msg.Name+"/"+msg.DeviceInfo.DevID, strMsg)
				switch msg.Name {
				case "existinglightchannelfound":
					fmt.Println("  **** "+msg.Name+" Webbrick updated - DEV ID is ", msg.DeviceInfo.DevID, " value of ", strconv.Itoa(int(msg.DeviceInfo.Level)))
				case "existingwebbrickupdated", "existingtriggerupdated", "existingbuttonupdated":
					fmt.Println("  **** "+msg.Name+" Webbrick updated! DEV ID is", msg.DeviceInfo.DevID)
					//fallthrough
				case "newlightchannelfound":
					fmt.Println("  **** "+msg.Name+" Webbrick found! DEV ID is ", msg.DeviceInfo.DevID, " value of ", strconv.Itoa(int(msg.DeviceInfo.Level)))
				case "newwebbrickfound", "newtriggerfound", "newbuttonfound":
					fmt.Println("  **** "+msg.Name+" Webbrick found! DEV ID is", msg.DeviceInfo.DevID)
					webbrick.PollWBStatus(msg.DeviceInfo.DevID)
				case "queried":
					// fmt.Println("We've queried. Name is:", msg.DeviceInfo.Name)
					// webbrick.SetState(msg.DeviceInfo.DevID, true)
					// time.Sleep(time.Second)
					// webbrick.SetState(msg.DeviceInfo.DevID, false)
				case "statechanged":
					fmt.Println("State changed to", msg.DeviceInfo.State)
				}
			default:
				webbrick.CheckForMessages()
			}
			// Wait for receiving a signal.
			//<-sigc

			// Disconnect the Network Connection.
			// if err := cli.Disconnect(); err != nil {
			// 	panic(err)
			// }

		}
		fmt.Println(" **** List of Devices ****")
		webbrick.ListDevices()
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

func publishMessage(cli *client.Client, message string, topic string) (bool, error) {

	var err error

	fmt.Println(" **** Called in publisher ****")

	// check that we have the topic set
	if topic == "" {
		topic = "webbrick"
	}
	//publish
	fmt.Println(" **** try to publish ****")
	err = cli.Publish(&client.PublishOptions{
		QoS:       mqtt.QoS0,
		TopicName: []byte(topic),
		Message:   []byte(message),
	})
	//check for err
	if err != nil {
		return false, err
		panic(err)
	}
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
