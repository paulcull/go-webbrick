package webbrick

// go-webbrck is a lightweight package that is used to control a variety the legacy webbrick products

import (
	"bytes"
	"code.google.com/p/go-charset/charset" // For XML conversion
	_ "code.google.com/p/go-charset/data"  // Specs for dataset conversion
	"encoding/xml"                         // For XML work
	"errors"                               // For crafting our own errors
	"fmt"                                  // For outputting stuff
	"github.com/davecgh/go-spew/spew"      // For neatly outputting stuff
	"io/ioutil"                            // HTTP body response processing
	"net"                                  // For networking stuff - for UDP
	"net/http"                             // For web http calls
	"reflect"                              // Type Get
	"strconv"                              // For String construction
	"time"                                 // For Poller
)

// EventStruct is our equivalent to node.js's Emitters, of sorts.
// This basically passes back to our Event channel, info about what event was raised
// (e.g. Device, plus an event name) so we can act appropriately
type EventStruct struct {
	Name       string
	DeviceInfo Device
}

// Device is info about the type of device that's been detected (socket, allone etc.)
type Device struct {
	ID          int    // The ID of our device
	DevID       string // The full Device ID
	Name        string // The name of our item
	Type        int    // What type of device this is. See the const below for valid types
	Channel     int    // Which Device Channel
	IP          net.IP // The IP address of our item
	Subscribed  bool   // Have we subscribed to this item yet? Doing so lets us control
	Queried     bool   // Have we queried this item for it's name and details yet?
	State       bool   // Is the item turned on or off? Will always be "false" for the AllOne, which doesn't do states, just IR & 433
	Level       int    // What is the level of the device
	LastMessage string // The last message to come through for this device
}

//////////////////////////////////
//
// Structure for UDP Message
//
//////////////////////////////////

// Webbrick message format
type WebBrickMsg struct {
	Addr          string
	PacketType    string
	PacketSource  string
	SourceChannel int
	TargetChannel int
	FromNodeNo    int
	ProcessMsg    bool
	Hour          string
	Minute        string
	Second        string
	Day           string
	Value         string
}

//////////////////////////////////
//
// Structure for WbStatus
//
//////////////////////////////////

type WebbrickStatus struct {
	Version    string `xml:"Ver,attr"`
	Error      int
	Context    int
	LoginState int
	BrickNo    int `xml:"SN"`
	DI         int
	DO         int
	Clock      Clock
	OWBus      int
	Tmps       struct{ Tmp []Tmp }
	AOs        struct{ AO []AO }
	AIs        struct{ AI []AI }
}

type Tmp struct {
	Id    int `xml:"id,attr"`
	Low   int `xml:"lo,attr"`
	High  int `xml:"hi,attr"`
	Value int `xml:",chardata"`
}

type AO struct {
	Id    int `xml:"id,attr"`
	Value int `xml:",chardata"`
}

type AI struct {
	Id    int `xml:"id,attr"`
	Low   int `xml:"lo,attr"`
	High  int `xml:"hi,attr"`
	Value int `xml:",chardata"`
}

type Clock struct {
	Date, Time string
	Day        int
}

//////////////////////////////////
//
// Structure for WbConfig
//
//////////////////////////////////

type WebbrickConfig struct {
	Version string `xml:"Ver,attr"`
	Name    string `xml:"NN"`
	IP      IPMac  `xml:"SI"`
	CDs     struct{ CD []CD }
	CTs     struct{ CT []CT }
	CIs     struct{ CI []CI }
	NOs     struct{ NO []NO }
	NAs     struct{ NA []NA }
}

type IPMac struct {
	IPString  string `xml:"ip,attr"`
	MACString string `xml:"mac,attr"`
}

// Device Configs
type CD struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
	Opt  int    `xml:"Opt,attr"`
	Trg  Trg
}

type CT struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
	TrgL TrgL
	TrgH TrgH
}

type CI struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
	TrgL TrgL
	TrgH TrgH
}

type NO struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
}

type NA struct {
	Id   int    `xml:"id,attr"`
	Name string `xml:"Name,attr"`
}

// Trigger Types
type Trg struct {
	B1 int `xml:"B1,attr"`
	B2 int `xml:"B2,attr"`
	B3 int `xml:"B3,attr"`
	B4 int `xml:"B4,attr"`
}

type TrgL struct {
	Lo int `xml:"Lo,attr"`
	B1 int `xml:"B1,attr"`
	B2 int `xml:"B2,attr"`
	B3 int `xml:"B3,attr"`
	B4 int `xml:"B4,attr"`
}

type TrgH struct {
	Hi int `xml:"Hi,attr"`
	B1 int `xml:"B1,attr"`
	B2 int `xml:"B2,attr"`
	B3 int `xml:"B3,attr"`
	B4 int `xml:"B4,attr"`
}

//////////////////////////////////
//
// Constants
//
//////////////////////////////////

const (
	UNKNOWN = -1 + iota // UNKNOWN is obviously a device that isn't implemented or is unknown. iota means add 1 to the next const, so SOCKET = 0, ALLONE = 1 etc.

	LIGHT     // LIGHT - is possibly a dimmer
	PIR       // PIR - Trigger
	BUTTON    // Pushbutton - Trigger
	TEMP      // Temp sensor
	STATE     // State
	HEARTBEAT // Heartbeat
)

// exlcusion := {""}

// pirs := {""}

//////////////////////////////////
//
// Core Structures
//
//////////////////////////////////

var conn *net.UDPConn // UDP Connection
var DEBUG = false
var POLL = true

const PollingTime = 30 // seconds

var Events = make(chan EventStruct, 1) // Events is our events channel which will notify calling code that we have an event happening
var Devices = make(map[string]*Device) // All the Devices we've discovered
var deviceCount int                    // How many items we've discovered

var UDPPort = "2552" // UDP Port

var gwURL = "home.pkhome.co.uk" // Gateway
var gwPORT = "8080"             // Gateway

// Our UDP connection

// ===============
// Exported Events
// ===============

// Prepare is the first function you should call. Gets our UDP connection ready
func Prepare() (bool, error) {

	_, err := getLocalIP() // Get our local IP. Not actually used in this func, but is more of a failsafe
	if err != nil {        // Error? Return false
		return false, err
	}

	udpAddr, resolveErr := net.ResolveUDPAddr("udp4", ":"+UDPPort) // Get our address ready for listening
	if resolveErr != nil {
		return false, resolveErr
	}

	var listenErr error
	conn, listenErr = net.ListenUDP("udp", udpAddr) // Now we listen on the address we just resolved
	if listenErr != nil {
		return false, listenErr
	}

	return true, nil
}

// ListDevices spews out info about all the Devices we know about. It's great because it includes counts and other stuff
func ListDevices() {
	spew.Dump(&Devices)
}

// CheckForMessages does what it says on the tin -- checks for incoming UDP messages
func CheckForMessages() (bool, error) { // Now we're checking for messages

	var msg []byte
	var buf [16]byte // We want to get 16 bytes of messages (is this enough? Need to check!)

	var success bool
	var err error

	n, addr, _ := conn.ReadFromUDP(buf[0:]) // Read 16 bytes from the buffer
	ip, _ := getLocalIP()                   // Get our local IP
	if n > 0 && addr.IP.String() != ip {    // If we've got more than 0 bytes and it's not from us

		msg = buf[0:n]                          // n is how many bytes we grabbed from UDP
		success, err = handleMessage(msg, addr) // Hand it off to our handleMessage func. We pass on the message and the address (for replying to messages)
		msg = nil                               // Clear out our msg property so we don't run handleMessage on old data

	} else {

		fmt.Println("From Us:", msg)
		msg = nil

	}

	return success, err
}

// Poller for getting Status on WB's in one go
func PollWBStatus(devID string) (int, error) {

	for _ = range time.Tick(PollingTime * time.Second) {
		//success, err := GetWBStatus(devID)
		GetWBStatus(devID)
	}

	return 1, nil

}

// Get WB Status on Initilisation
func GetWBStatus(devID string) (int, error) {

	if !DEBUG {
		fmt.Println("   **** Getting WBStatus & Config for ", devID)
	}

	var success int
	var statusCommand string
	var configCommand string
	// statusCommand = "http://" + gwURL + ":" + gwPORT + "/wbproxy/" + Devices[devID].IP.String() + "/WbStatus.xml"
	// configCommand = "http://" + gwURL + ":" + gwPORT + "/wbproxy/" + Devices[devID].IP.String() + "/WbCfg.xml"
	statusCommand = "http://" + Devices[devID].IP.String() + "/WbStatus.xml"
	configCommand = "http://" + Devices[devID].IP.String() + "/WbCfg.xml"

	///////////////////////////////
	//
	// WB Status
	//
	////////////////////////////////

	// http call for the wb status
	resp, err := http.Get(statusCommand) // call the http service
	if err != nil {
		fmt.Println("Error getting WBStatus for " + Devices[devID].IP.String())
	}
	defer resp.Body.Close()

	respbody, err := ioutil.ReadAll(resp.Body) // read out the reponsse body
	if err != nil {
		fmt.Println("Empty Body in http request for " + Devices[devID].IP.String())
	} else {
		success = 1
	}
	//fmt.Printf("%s \n", respbody)

	// Decode WB Status XML encoding
	var _wbs WebbrickStatus                   // create container to load the xml
	reader := bytes.NewReader(respbody)       // create a new reader for transcoding to utf-8
	decoder := xml.NewDecoder(reader)         // create a new xml decoder
	decoder.CharsetReader = charset.NewReader // bind the reader to the decoder
	xmlerr := decoder.Decode(&_wbs)           // unmarshall the xml
	if xmlerr != nil {
		fmt.Printf("error: %v", xmlerr)
		//return
	} else {
		fmt.Println("      **** Got WebbrickStatus ok for ", devID)

	}
	//fmt.Printf("%+v\n", _wbs)

	///////////////////////////////
	//
	// WB Config
	//
	////////////////////////////////

	// http call for the wb config
	wbcresp, wbcerr := http.Get(configCommand) // call the http service
	if wbcerr != nil {
		fmt.Println("Error getting WBConfig for " + Devices[devID].IP.String())
	}
	defer wbcresp.Body.Close()

	wbcrespbody, wbcerr := ioutil.ReadAll(wbcresp.Body) // read out the reponsse body
	if wbcerr != nil {
		fmt.Println("Empty Body in http request for " + Devices[devID].IP.String())
		success = 0
		err = wbcerr
	} else {
		success = 1
	}
	//fmt.Printf("%s \n", wbcrespbody)

	// Decode WB Config XML encoding
	var _wbc WebbrickConfig                      // create container to load the xml
	wbcreader := bytes.NewReader(wbcrespbody)    // create a new reader for transcoding to utf-8
	wbcdecoder := xml.NewDecoder(wbcreader)      // create a new xml decoder
	wbcdecoder.CharsetReader = charset.NewReader // bind the reader to the decoder
	wbcxmlerr := wbcdecoder.Decode(&_wbc)        // unmarshall the xml
	if wbcxmlerr != nil {
		fmt.Printf("error: %v", wbcxmlerr)
		//return
	} else {
		fmt.Println("      **** Got WebbrickConfig ok for ", _wbc.Name)
		success = 0
		err = wbcxmlerr
	}
	//fmt.Printf("%+v\n", _wbc)

	mapDevices, mderr := CreateBrickDevices(_wbc, _wbs)

	if mderr != nil {
		fmt.Println("error mapping devices %v", mderr)
		success = mapDevices
		err = mderr
	}

	if DEBUG {
		ListDevices()
	}

	return success, err
}

///////////////////////////////////////////
//
// Creating for the new devices
//
///////////////////////////////////////////

func CreateBrickDevices(_wbc WebbrickConfig, _wbs WebbrickStatus) (int, error) {

	var success int
	var err error
	var _ip net.IP

	success = 1
	err = nil

	_ip = net.ParseIP(_wbc.IP.IPString)

	fmt.Println("      **** Creating Devices for ", _wbc.Name)

	// Lights - AO
	for light := range _wbs.AOs.AO {

		var _state bool
		var _message string

		// // Calculate the UID
		UID := strconv.Itoa(_wbs.BrickNo) + "::AO::" + strconv.Itoa(light)
		if _wbs.AOs.AO[light].Value > 0 {
			_state = false
			_message = _wbc.NAs.NA[light].Name + " is off"
		} else {
			_state = true
			_message = _wbc.NAs.NA[light].Name + " is on at " + strconv.Itoa(_wbs.AOs.AO[light].Value) + "%"
		}

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", LIGHT, _wbs.AOs.AO[light].Id, _ip, true, true, _state, _wbs.AOs.AO[light].Value, _message}
			passMessage("newlightchannelfound", *Devices[UID])
			fmt.Println("        **** Creating Light Device for ", UID, _wbs.AOs.AO[light], _wbc.NAs.NA[light])
		} else {
			Devices[UID].State = _state
			Devices[UID].Level = _wbs.AOs.AO[light].Value
			Devices[UID].LastMessage = _message
			passMessage("existinglightchannelupdated", *Devices[UID])
			fmt.Println("        **** Updating Light Device for ", UID, _wbs.AOs.AO[light], _wbc.NAs.NA[light])
		}

	}

	//Buttons  & PIR
	for digitalIn := range _wbc.CDs.CD {

		var _message string
		_message = _wbc.CDs.CD[digitalIn].Name + " has been pressed"

		// // Calculate the UID
		UID := strconv.Itoa(_wbs.BrickNo) + "::TD::" + strconv.Itoa(digitalIn)

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", BUTTON, digitalIn, _ip, true, true, false, 0, _message}
			passMessage("newbuttonfound", *Devices[UID])
			fmt.Println("        **** Creating Button Device for ", UID, _wbc.CDs.CD[digitalIn])
		} else {
			Devices[UID].LastMessage = _message
			passMessage("existingbuttonupdated", *Devices[UID])
			fmt.Println("        **** Updating Button Device for ", UID, _wbc.CDs.CD[digitalIn])
		}

	}

	// Hardware State
	for digitalOut := range _wbc.NOs.NO {

		var _message string
		_message = _wbc.NOs.NO[digitalOut].Name + " state has changed"

		// // Calculate the UID
		UID := strconv.Itoa(_wbs.BrickNo) + "::TD::" + strconv.Itoa(digitalOut)

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", STATE, digitalOut, _ip, true, true, false, 0, _message}
			passMessage("newoutputfound", *Devices[UID])
			fmt.Println("        **** Creating State Device for ", UID, _wbc.NOs.NO[digitalOut])
		} else {
			Devices[UID].LastMessage = _message
			passMessage("existingoutputupdated", *Devices[UID])
			fmt.Println("        **** Updating State Device for ", UID, _wbc.NOs.NO[digitalOut])
		}

	}

	// Temps State
	for temp := range _wbc.CTs.CT {

		var _message string
		_message = _wbc.CTs.CT[temp].Name + " temperature value has changed to " + strconv.Itoa(_wbs.Tmps.Tmp[temp].Value)

		// // Calculate the UID
		UID := strconv.Itoa(_wbs.BrickNo) + "::TD::" + strconv.Itoa(temp)

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", STATE, temp, _ip, true, true, false, _wbs.Tmps.Tmp[temp].Value, _message}
			passMessage("newtempfound", *Devices[UID])
			fmt.Println("        **** Creating Temperature Device for ", UID, _wbc.CTs.CT[temp])
		} else {
			Devices[UID].LastMessage = _message
			passMessage("existingtempupdated", *Devices[UID])
			fmt.Println("        **** Updating Temperature Device for ", UID, _wbc.CTs.CT[temp])
		}

	}

	return success, err

}

///////////////////////////////////////////
//
// Functions for actioning
//
///////////////////////////////////////////

// ToggleState finds out if the socket is on or off, then toggles it
func ToggleState(devID string) (bool, error) {
	if Devices[devID].State == true {
		return SetState(devID, false)
	}

	return SetState(devID, true)
}

// SetState sets the state of a socket, given its MAC address
func SetState(devID string, state bool) (bool, error) {

	var command string

	switch Devices[devID].Type {

	// Its a light
	case LIGHT:
		// update the record for new levels
		Devices[devID].State = state
		//var statebit string
		if state == true {
			Devices[devID].Level = 100
		} else {
			Devices[devID].Level = 0
		}

		// create and send the command
		command = "http://" + Devices[devID].IP.String() + "/hid.spi?AA" + strconv.Itoa(Devices[devID].Channel) + "=" + strconv.Itoa(Devices[devID].Level)
		success, err := sendCommand(command, devID)

		//success, err := sendMessage("686400176463"+macAdd+twenties+"00000000"+statebit, Devices[macAdd].IP)
		passMessage("lightset:"+strconv.Itoa(Devices[devID].Level), *Devices[devID])
		command = ""
		return success, err

	// Its a fake button to trigger the DI
	case BUTTON:

		// create and send the command
		command = "http://" + Devices[devID].IP.String() + "/hid.spi?DI" + strconv.Itoa(Devices[devID].Channel)
		success, err := sendCommand(command, devID)

		//success, err := sendMessage("686400176463"+macAdd+twenties+"00000000"+statebit, Devices[macAdd].IP)
		passMessage("button", *Devices[devID])
		command = ""
		return success, err

	// don't know what to do or how to do it
	default:
		command = ""
		return false, errors.New("Can't set state on " + strconv.Itoa(Devices[devID].Type))
	}

}

// GetState sets the state of a socket, given its MAC address
func GetState(devID string) (int, error) {

	//var command string

	switch Devices[devID].Type {

	// // Its a light
	// case LIGHT:
	// 	// create and send the command
	// 	command = "http://" + Devices[devID].IP.String() + "/hid.spi?AA" + strconv.Itoa(Devices[devID].Channel) + "=" + strconv.Itoa(Devices[devID].Level)
	// 	success, err := sendCommand(command, devID)
	// 	//success, err := sendMessage("686400176463"+macAdd+twenties+"00000000"+statebit, Devices[macAdd].IP)
	// 	passMessage("lightset:"+strconv.Itoa(Devices[devID].Level), *Devices[devID])
	// 	command = ""
	// 	return success, err

	// // Its a fake button to trigger the DI
	// case BUTTON:
	// 	// create and send the command
	// 	command = "http://" + Devices[devID].IP.String() + "/hid.spi?DI" + strconv.Itoa(Devices[devID].Channel)
	// 	success, err := sendCommand(command, devID)
	// 	//success, err := sendMessage("686400176463"+macAdd+twenties+"00000000"+statebit, Devices[macAdd].IP)
	// 	passMessage("button", *Devices[devID])
	// 	command = ""
	// 	return success, err

	// // don't know what to do or how to do it
	default:
		//command = ""
		return 0, errors.New("Can't set state on " + strconv.Itoa(Devices[devID].Type))
	}

}

// ==================
// Internal functions
// ==================

// handleMessage parses a message found by CheckForMessages
func handleMessage(buf []byte, addr *net.UDPAddr) (bool, error) {

	//Strip out the information sent from the brick
	resp := new(WebBrickMsg)
	resp.Addr = addr.IP.String()

	for index, element := range buf {
		switch index {
		case 1:
			resp.PacketType = string(element)
		case 2:
			resp.PacketSource = string(element)
		case 3:
			resp.PacketSource = resp.PacketSource + string(element)
		case 4:
			switch resp.PacketSource {
			case "ST": // Handle time message differently
				resp.Hour = strconv.Itoa(int(element))
			default:
				//resp.SourceChannel = strconv.Itoa(int(element))
				resp.SourceChannel = int(element)
			}
		case 5:
			switch resp.PacketSource {
			case "ST": // Handle time message differently
				resp.Minute = strconv.Itoa(int(element))
			default:
				//resp.TargetChannel = strconv.Itoa(int(element))
				resp.TargetChannel = int(element)
			}
		case 6:
			switch resp.PacketSource {
			case "ST": // Handle time message differently
				resp.Second = strconv.Itoa(int(element) / 2)
			default:
			}
		case 7:
			resp.FromNodeNo = int(element)
		case 9:
			switch resp.PacketSource {
			case "ST": // Handle time message differently
				resp.Day = strconv.Itoa(int(element))
			default:
			}
		case 11:
			switch resp.PacketSource {
			case "AO", "AI":
				resp.Value = strconv.Itoa(int(element))
			default:
			}
		default:

		}

		// index is the index where we are
		// element is the element from someSlice for where we are
		if DEBUG && index > 3 && resp.PacketSource != "ST" && resp.PacketSource != "AO" && resp.PacketSource != "DO" && resp.PacketSource != "TD" {
			fmt.Println("Missing Handler for " + resp.PacketSource)
			fmt.Println(index)
			fmt.Printf(" : ")
			fmt.Println((reflect.TypeOf(element)))
			fmt.Printf(" : ")
			fmt.Println((element))
			fmt.Printf(" : ")
			fmt.Println(string(element))
			fmt.Printf("\n")
		}
	}

	if DEBUG {
		fmt.Printf("\n :::: ")
		fmt.Printf(resp.Addr)
		fmt.Printf(" :::: ")
		fmt.Printf(strconv.Itoa(resp.FromNodeNo))
		fmt.Printf(" ::--:: ")
		fmt.Printf(resp.PacketSource)
		fmt.Printf(" :::: ")
		fmt.Printf(strconv.Itoa(resp.SourceChannel))
		fmt.Printf(" :::: ")
		fmt.Printf(resp.Value)
		fmt.Printf(" ::--:: ")
		fmt.Printf(resp.Hour)
		fmt.Printf(" :::: ")
		fmt.Printf(resp.Minute)
		fmt.Printf(" :::: ")
		fmt.Printf(resp.Second)
		fmt.Printf(" :::: ")
		fmt.Printf(resp.Day)
		fmt.Printf(" :::: \n")
		fmt.Print(resp)
		fmt.Print("\n")
	}

	UID := strconv.Itoa(resp.FromNodeNo) + "::" + resp.PacketSource + "::" + strconv.Itoa(resp.SourceChannel)

	switch resp.PacketSource {
	case "ST": // Timestamp

		_message := "Seen at " + resp.Hour + ":" + resp.Minute + ":" + resp.Second

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", HEARTBEAT, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
			passMessage("newwebbrickfound", *Devices[UID])
		} else {
			Devices[UID].LastMessage = _message
			passMessage("existingwebbrickupdated", *Devices[UID])
		}

	case "DO": // State, e.g. Heating, State Tracking

		_message := "Trigger on " + strconv.Itoa(resp.SourceChannel)

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", PIR, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
			passMessage("newtriggerfound", *Devices[UID])
		} else {
			Devices[UID].LastMessage = _message
			passMessage("existingtriggerupdated", *Devices[UID])
		}

	case "TD": // Button (?) Check is this includes PIR as well

		_message := "Button on " + strconv.Itoa(resp.SourceChannel)

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", BUTTON, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
			passMessage("newbuttonfound", *Devices[UID])
		} else {
			Devices[UID].LastMessage = _message
			passMessage("existingbuttonupdated", *Devices[UID])
		}

	case "AO": // Light Dimmer Device

		// Calculate the private values for the message
		var _state bool
		_message := "Light at level " + resp.Value
		_value, _ := strconv.Atoi(resp.Value)
		if _value > 0 {
			_state = true
		} else {
			_state = false
		}

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", LIGHT, resp.SourceChannel, addr.IP, true, false, _state, _value, _message}
			passMessage("newlightchannelfound", *Devices[UID])
		} else {
			Devices[UID].State = _state
			Devices[UID].Level = _value
			Devices[UID].LastMessage = _message
			passMessage("existinglightchannelupdated", *Devices[UID])
		}
	}
	return true, nil
}

// handleMessage parses a message found by CheckForMessages
// func handleWB(xml string, addr *net.UDPAddr) (bool, error) {

// 	//Strip out the information sent from the brick
// 	resp := new(WebBrickMsg)
// 	resp.Addr = addr.IP.String()

// 	UID := strconv.Itoa(resp.FromNodeNo) + "::" + resp.PacketSource + "::" + strconv.Itoa(resp.SourceChannel)

// 	switch resp.PacketSource {
// 	case "ST": // Timestamp

// 		_message := "Seen at " + resp.Hour + ":" + resp.Minute + ":" + resp.Second

// 		// Check to see if we've already got macAdd in our array
// 		_, ok := Devices[UID]

// 		if ok == false { // we haven't got this in our Devices array
// 			deviceCount++
// 			Devices[UID] = &Device{deviceCount, UID, "", HEARTBEAT, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
// 			passMessage("newwebbrickfound", *Devices[UID])
// 		} else {
// 			Devices[UID].LastMessage = _message
// 			passMessage("existingwebbrickupdated", *Devices[UID])
// 		}

// 	case "DO": // Is this PIR or is this Dig Out State, e.g. Heating, State Tracking ?

// 		_message := "Trigger on " + strconv.Itoa(resp.SourceChannel)

// 		// Check to see if we've already got macAdd in our array
// 		_, ok := Devices[UID]

// 		if ok == false { // we haven't got this in our Devices array
// 			deviceCount++
// 			Devices[UID] = &Device{deviceCount, UID, "", PIR, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
// 			passMessage("newtriggerfound", *Devices[UID])
// 		} else {
// 			Devices[UID].LastMessage = _message
// 			passMessage("existingtriggerupdated", *Devices[UID])
// 		}

// 	case "TD": // Button (?) Check is this includes PIR as well

// 		_message := "Button on " + strconv.Itoa(resp.SourceChannel)

// 		// Check to see if we've already got macAdd in our array
// 		_, ok := Devices[UID]

// 		if ok == false { // we haven't got this in our Devices array
// 			deviceCount++
// 			Devices[UID] = &Device{deviceCount, UID, "", BUTTON, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
// 			passMessage("newbuttonfound", *Devices[UID])
// 		} else {
// 			Devices[UID].LastMessage = _message
// 			passMessage("existingbuttonupdated", *Devices[UID])
// 		}

// 	case "AO": // Light Dimmer Device

// 		// Calculate the private values for the message
// 		var _state bool
// 		_message := "Light at level " + resp.Value
// 		_value, _ := strconv.Atoi(resp.Value)
// 		if _value > 0 {
// 			_state = true
// 		} else {
// 			_state = false
// 		}

// 		// Check to see if we've already got macAdd in our array
// 		_, ok := Devices[UID]

// 		if ok == false { // we haven't got this in our Devices array
// 			deviceCount++
// 			Devices[UID] = &Device{deviceCount, UID, "", LIGHT, resp.SourceChannel, addr.IP, true, false, _state, _value, _message}
// 			passMessage("newlightchannelfound", *Devices[UID])
// 		} else {
// 			Devices[UID].State = _state
// 			Devices[UID].Level = _value
// 			Devices[UID].LastMessage = _message
// 			passMessage("existinglightchannelupdated", *Devices[UID])
// 		}
// 	}
// 	return true, nil
// }

////////////////////////////////////
//
//  Local Helper Functions
//
////////////////////////////////////

// Gets our current IP address. This is used so we can ignore messages from ourselves
func getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	ifaces, _ := net.Interfaces()
	// handle err
	for _, i := range ifaces {
		addrs, _ := i.Addrs()
		// handle err
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPAddr:
				return v.IP.String(), nil
			}

		}
	}

	return "", errors.New("Unable to find IP address. Ensure you're connected to a network")
}

// passMessage adds items to our Events channel so the calling code can be informed
// It's non-blocking or whatever.
func passMessage(message string, device Device) bool {

	select {
	case Events <- EventStruct{message, device}:

	default:
	}

	return true
}

// sendCommand is the key instruction part of the library
//		success, err := sendCommand(command, devID)
func sendCommand(command string, devID string) (bool, error) {

	resp, err := http.Get(command)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Dont need to read the response yet
	// its fire and hope and forget
	body, err := ioutil.ReadAll(resp.Body)
	fmt.Println("**** Body Message ****", body)

	// Looks good
	return true, nil

}
