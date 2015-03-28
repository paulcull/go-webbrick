package main

import (
	"fmt"                             // For outputting messages
	"github.com/paulcull/go-webbrick" // For controlling Orvibo stuff
	"strconv"
	"time" // For setInterval()
)

func main() {
	fmt.Println("**** Starting Test...")
	ready, err := webbrick.Prepare() // You ready?
	if ready == true {               // Yep! Let's do this!

		for { // Loop forever
			select { // This lets us do non-blocking channel reads. If we have a message, process it. If not, check for UDP data and loop
			case msg := <-webbrick.Events:
				fmt.Println(" **** Event for ", msg.Name, "received...")
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
					//orvibo.Subscribe() // Subscribe to any unsubscribed sockets
					//orvibo.Query()     // And query any unqueried sockets
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

		}
		fmt.Println(" **** List of Devices ****")
		webbrick.ListDevices()
	} else {
		fmt.Println("Error:", err)
	}

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
