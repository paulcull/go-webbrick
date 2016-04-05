package webbrick

// go-webbrck is a lightweight package that is used to control a variety the legacy webbrick products

import (
	"bytes"
	"github.com/paulrosania/go-charset/charset" // For XML conversion
	_ "github.com/paulrosania/go-charset/data"  // Specs for dataset conversion
	"encoding/xml"                         // For XML work
	"errors"                               // For crafting our own errors
	"fmt"                                  // For outputting stuff
	"github.com/davecgh/go-spew/spew"      // For neatly outputting stuff
	"github.com/juju/loggo"                //  logging
	"io/ioutil"                            // HTTP body response processing
	"net"                                  // For networking stuff - for UDP
	"net/http"                             // For web http calls
	"reflect"                              // Type Get
	"strconv"                              // For String construction
	"strings"                              // for Upper case conversion
	"time"                                 // For Poller
)

var myLog = loggo.GetLogger("Webbrick")

// EventStruct is our equivalent to node.js's Emitters, of sorts.
// This basically passes back to our Event channel, info about what event was raised
// (e.g. Device, plus an event name) so we can act appropriately
type WebbrickDriverConfig struct {
	Name string
	//	NinjaLogControl *logger.Logger
	Initialised     bool
	NumberOfDevices int
	PollingMinutes  int
	PollingActive   bool
}

// EventStruct is our equivalent to node.js's Emitters, of sorts.
// This basically passes back to our Event channel, info about what event was raised
// (e.g. Device, plus an event name) so we can act appropriately
type EventStruct struct {
	Name       string
	DeviceInfo Device
}

// Device is info about the type of device that's been detected (socket, allone etc.)
type Device struct {
	ID          int     // The ID of our device
	DevID       string  // The full Device ID
	Name        string  // The name of our item
	BrickID     int     // The ID for the brick unit
	Type        int     // What type of device this is. See the const below for valid types
	Channel     int     // Which Device Channel
	IP          net.IP  // The IP address of our item
	Subscribed  bool    // Have we subscribed to this item yet? Doing so lets us control
	Queried     bool    // Have we queried this item for it's name and details yet?
	State       bool    // Is the item turned on or off? Will always be "false" for the AllOne, which doesn't do states, just IR & 433
	Level       float64 // What is the level of the device
	LastMessage string  // The last message to come through for this device
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
	Id    int     `xml:"id,attr"`
	Low   int     `xml:"lo,attr"`
	High  int     `xml:"hi,attr"`
	Value float64 `xml:",chardata"`
}

type AO struct {
	Id    int     `xml:"id,attr"`
	Value float64 `xml:",chardata"`
}

type AI struct {
	Id    int     `xml:"id,attr"`
	Low   int     `xml:"lo,attr"`
	High  int     `xml:"hi,attr"`
	Value float64 `xml:",chardata"`
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

// Setup some lists for a set of PIR's
var PIRS = map[string]bool{
	"2::TD::0":  true,
	"2::TD::2":  true,
	"2::TD::11": true,
	"2::TD::1":  true,
}

// Setup some lists for Devices on the webbricks that aren't in use
var EXCLUDE = map[string]bool{
	"2::DO::1": true, "2::DO::2": true, "2::DO::3": true, "2::DO::4": true, "2::DO::5": true, "2::DO::6": true, "2::DO::7": true,
	"2::AO::1": true,
	"2::CT::3": true, "2::CT::4": true,
	"3::DO::0": true, "3::DO::1": true, "3::DO::2": true, "3::DO::3": true, "3::DO::4": true, "3::DO::5": true, "3::DO::6": true, "3::DO::7": true,
	"3::AO::1": true,
	"3::CT::0": true, "3::CT::1": true, "3::CT::2": true, "3::CT::3": true, "3::CT::4": true,
	"4::DO::0": true, "4::DO::1": true, "4::DO::2": true, "4::DO::3": true, "4::DO::4": true, "4::DO::5": true, "4::DO::6": true, "4::DO::7": true,
	"4::AO::0": true,
	"4::CT::1": true, "4::CT::2": true, "4::CT::3": true, "4::CT::4": true,
	"5::TD::1": true, "5::TD::2": true, "5::TD::3": true, "5::TD::4": true,
	"5::CT::1": true, "5::CT::2": true, "5::CT::3": true, "5::CT::4": true,
	"6::DO::0": true, "6::DO::1": true, "6::DO::2": true, "6::DO::3": true, "6::DO::4": true, "6::DO::5": true, "6::DO::6": true, "6::DO::7": true,
	"6::AO::0": true,
	"6::CT::2": true, "6::CT::3": true, "6::CT::4": true,
	"7::DO::0": true, "7::DO::1": true, "7::DO::2": true, "7::DO::3": true, "7::DO::4": true, "7::DO::5": true, "7::DO::6": true, "7::DO::7": true,
	"7::CT::1": true, "7::CT::2": true, "7::CT::3": true, "7::CT::4": true,
}

//////////////////////////////////
//
// Core Structures
//
//////////////////////////////////

var conn *net.UDPConn // UDP Connection
var DEBUG = false
var POLL = false
var PollingMinutes int

const PollingTime = 600

var Events = make(chan EventStruct, 50) // Events is our events channel which will notify calling code that we have an event happening
var Devices = make(map[string]*Device)  // All the Devices we've discovered
var deviceCount int                     // How many items we've discovered

var UDPPort = "2552" // UDP Port

var gwURL = "home.pkhome.co.uk" // Gateway
var gwPORT = "8080"             // Gateway

// Our UDP connection

// ===============
// Exported Events
// ===============

// Prepare is the first function you should call. Gets our UDP connection ready
func Prepare(wbdc *WebbrickDriverConfig) (bool, error) {

	if wbdc != nil {
		wbdc =
			&WebbrickDriverConfig{
				Name:        "PKHome-TEST",
				Initialised: false,
				//				NinjaLogControl: logger.GetLogger("WebBrick Local"),
				NumberOfDevices: 0,
				PollingMinutes:  5,
				PollingActive:   false,
			}
	}

	//POLL = wbdc.PollingActive
	//PollingMinutes = wbdc.PollingMinutes

	//	myLog = wbdc.NinjaLogControl
	//myLog = loggo.GetLogger("WebBrick Local")

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

		myLog.Infof("From Us:", msg)
		msg = nil

	}

	return success, err
}

// Poller for getting Status on WB's in one go
func PollWBStatus(devID string) (int, error) {

	// Run straight away
	GetWBStatus(devID)
	// then run based on the interval

	if POLL {
		//myLog.Debugf("*************************** %s **************************", reflect.TypeOf(PollingTime))
		//		for _ = range time.Tick(PollingMinutes * time.Minute) {
		for _ = range time.Tick(PollingTime * time.Second) {
			myLog.Infof("   **** Polling WBStatus & Config for ", devID)
			GetWBStatus(devID)
		}
	}
	return 1, nil

}

// Get WB Status on Initilisation
func GetWBStatus(devID string) (int, error) {

	myLog.Infof("   **** Getting WBStatus & Config for ", devID)

	var success int
	var statusCommand string
	var configCommand string
	// will need to use the gateway if the call is outside the local network
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
		myLog.Errorf("Error getting WBStatus for " + Devices[devID].IP.String())
		return 0, err
	}
	defer resp.Body.Close()

	respbody, err := ioutil.ReadAll(resp.Body) // read out the reponsse body
	if err != nil {
		myLog.Errorf("Empty Body in http request for " + Devices[devID].IP.String())
		return 0, err
	} else {
		success = 1
	}
	//myLog.Debugf("%s \n", respbody)

	// Decode WB Status XML encoding
	var _wbs WebbrickStatus                   // create container to load the xml
	reader := bytes.NewReader(respbody)       // create a new reader for transcoding to utf-8
	decoder := xml.NewDecoder(reader)         // create a new xml decoder
	decoder.CharsetReader = charset.NewReader // bind the reader to the decoder
	xmlerr := decoder.Decode(&_wbs)           // unmarshall the xml
	if xmlerr != nil {
		myLog.Errorf("error: %v", xmlerr)
		return 0, xmlerr
	} else {
		myLog.Infof("      **** Got WebbrickStatus ok for ", devID)

	}

	if DEBUG {
		myLog.Debugf(spew.Sdump(_wbs))
	}
	///////////////////////////////
	//
	// WB Config
	//
	////////////////////////////////

	// http call for the wb config
	wbcresp, wbcerr := http.Get(configCommand) // call the http service
	if wbcerr != nil {
		myLog.Errorf("Error getting WBConfig for " + Devices[devID].IP.String())
	}
	defer wbcresp.Body.Close()

	wbcrespbody, wbcerr := ioutil.ReadAll(wbcresp.Body) // read out the reponsse body
	if wbcerr != nil {
		myLog.Errorf("Empty Body in http request for " + Devices[devID].IP.String())
		success = 0
		err = wbcerr
		return success, wbcerr
	} else {
		success = 1
	}
	//myLog.Debugf("%s \n", wbcrespbody)

	// Decode WB Config XML encoding
	var _wbc WebbrickConfig                      // create container to load the xml
	wbcreader := bytes.NewReader(wbcrespbody)    // create a new reader for transcoding to utf-8
	wbcdecoder := xml.NewDecoder(wbcreader)      // create a new xml decoder
	wbcdecoder.CharsetReader = charset.NewReader // bind the reader to the decoder
	wbcxmlerr := wbcdecoder.Decode(&_wbc)        // unmarshall the xml
	if wbcxmlerr != nil {
		myLog.Errorf("error: %v", wbcxmlerr)
		return 0, wbcxmlerr
	} else {
		myLog.Infof("      **** Got WebbrickConfig ok for ", _wbc.Name)
		success = 1
		//err = wbcxmlerr
	}

	if DEBUG {
		myLog.Debugf(spew.Sdump(_wbc))
	}

	mapDevices, mderr := CreateBrickDevices(_wbc, _wbs)

	if mderr != nil {
		myLog.Errorf("error mapping devices %v", mderr)
		success = mapDevices
		err = mderr
		return 0, mderr
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

	myLog.Infof("      **** Checking Devices for ", _wbc.Name)

	// Lights - AO
	for light := range _wbs.AOs.AO {

		var _state bool
		var _message string

		// // Calculate the UID
		UID := strconv.Itoa(_wbs.BrickNo) + "::AO::" + strconv.Itoa(light)
		if !EXCLUDE[UID] {
			if _wbs.AOs.AO[light].Value == 0 {
				_state = false
				_message = _wbc.NAs.NA[light].Name + " is off"
			} else {
				_state = true
				_message = _wbc.NAs.NA[light].Name + " is on at " + strconv.FormatFloat(_wbs.AOs.AO[light].Value, 'f', 6, 64) + "%"
			}

			// Check to see if we've already got macAdd in our array
			_, ok := Devices[UID]

			if ok == false { // we haven't got this in our Devices array
				deviceCount++
				Devices[UID] = &Device{deviceCount, UID, _wbc.NAs.NA[light].Name, _wbs.BrickNo, LIGHT, _wbs.AOs.AO[light].Id, _ip, true, true, _state, _wbs.AOs.AO[light].Value, _message}
				passMessage("newlightchannelfound", *Devices[UID])
				myLog.Infof("        **** Creating Light Device for ", UID, _wbs.AOs.AO[light], _wbc.NAs.NA[light])
			} else {
				Devices[UID].State = _state
				Devices[UID].Name = _wbc.NAs.NA[light].Name
				Devices[UID].Level = _wbs.AOs.AO[light].Value
				Devices[UID].LastMessage = _message
				passMessage("existinglightchannelupdated", *Devices[UID])
				myLog.Infof("        **** Updating Light Device for ", UID, _wbs.AOs.AO[light], _wbc.NAs.NA[light])
			}
		} else {
			myLog.Infof("        **** Excluding Light Device for ", UID)
		}
	}

	//Buttons  & PIR
	for digitalIn := range _wbc.CDs.CD {

		var _message string

		// // Calculate the UID
		UID := strconv.Itoa(_wbs.BrickNo) + "::TD::" + strconv.Itoa(digitalIn)

		// TODO : remove PIR exclusion and fix the channel that is should create
		if !EXCLUDE[UID] { //&& !PIRS[UID] {

			// Check to see if we've already got macAdd in our array
			_, ok := Devices[UID]

			if ok == false { // we haven't got this in our Devices array
				if !PIRS[UID] { // handle PIR from list, as you can't tell the difference normally
					_message = _wbc.CDs.CD[digitalIn].Name + " has been found"
					deviceCount++
					Devices[UID] = &Device{deviceCount, UID, _wbc.CDs.CD[digitalIn].Name, _wbs.BrickNo, BUTTON, digitalIn, _ip, true, true, false, 0, _message}
					passMessage("newbuttonfound", *Devices[UID])
					myLog.Infof("        **** Creating Button Device for ", UID, _wbc.CDs.CD[digitalIn])
				} else {
					_message = _wbc.CDs.CD[digitalIn].Name + " has been found"
					deviceCount++
					Devices[UID] = &Device{deviceCount, UID, _wbc.CDs.CD[digitalIn].Name, _wbs.BrickNo, PIR, digitalIn, _ip, true, true, false, 0, _message}
					passMessage("newpirfound", *Devices[UID])
					myLog.Infof("        **** Creating PIR Device for ", UID, _wbc.CDs.CD[digitalIn])
				}
			} else {
				if !PIRS[UID] {
					_message = _wbc.CDs.CD[digitalIn].Name + " has been pressed"
					Devices[UID].LastMessage = _message
					Devices[UID].Name = _wbc.CDs.CD[digitalIn].Name
					passMessage("existingbuttonupdated", *Devices[UID])
					myLog.Infof("        **** Updating Button Device for ", UID, _wbc.CDs.CD[digitalIn])
				} else {
					_message = _wbc.CDs.CD[digitalIn].Name + " has been triggered"
					Devices[UID].LastMessage = _message
					Devices[UID].Name = _wbc.CDs.CD[digitalIn].Name
					passMessage("existingpirupdated", *Devices[UID])
					myLog.Infof("        **** Updating PIR Device for ", UID, _wbc.CDs.CD[digitalIn])
				}
			}
		} else {
			myLog.Infof("        **** Excluding Trigger Device for ", UID)

		}
	}

	// Hardware State
	for digitalOut := range _wbc.NOs.NO {

		var _message string

		// // Calculate the UID
		UID := strconv.Itoa(_wbs.BrickNo) + "::DO::" + strconv.Itoa(digitalOut)

		if !EXCLUDE[UID] {
			// Check to see if we've already got macAdd in our array
			_, ok := Devices[UID]

			if ok == false { // we haven't got this in our Devices array
				deviceCount++
				_message = _wbc.NOs.NO[digitalOut].Name + " state has been found"
				Devices[UID] = &Device{deviceCount, UID, _wbc.NOs.NO[digitalOut].Name, _wbs.BrickNo, STATE, digitalOut, _ip, true, true, false, 0, _message}
				passMessage("newoutputfound", *Devices[UID])
				myLog.Infof("        **** Creating State Device for ", UID, _wbc.NOs.NO[digitalOut])
			} else {
				_message = _wbc.NOs.NO[digitalOut].Name + " state has changed"
				Devices[UID].LastMessage = _message
				Devices[UID].Name = _wbc.NOs.NO[digitalOut].Name
				passMessage("existingoutputupdated", *Devices[UID])
				myLog.Infof("        **** Updating State Device for ", UID, _wbc.NOs.NO[digitalOut])
			}
		} else {
			myLog.Infof("        **** Excluding State Device for ", UID)

		}
	}

	// Temps State
	for temp := range _wbc.CTs.CT {

		var _message string
		_message = _wbc.CTs.CT[temp].Name + " temperature value is " + strconv.FormatFloat((_wbs.Tmps.Tmp[temp].Value/16), 'f', 2, 64)

		// Calculate the UID
		UID := strconv.Itoa(_wbs.BrickNo) + "::CT::" + strconv.Itoa(temp)

		if !EXCLUDE[UID] {
			// Check to see if we've already got macAdd in our array
			_, ok := Devices[UID]

			if ok == false { // we haven't got this in our Devices array
				deviceCount++
				Devices[UID] = &Device{deviceCount, UID, _wbc.CTs.CT[temp].Name, _wbs.BrickNo, TEMP, temp, _ip, true, true, false, (_wbs.Tmps.Tmp[temp].Value / 16), _message}
				passMessage("newtempfound", *Devices[UID])
				myLog.Infof("        **** Creating Temperature Device for ", UID, _wbc.CTs.CT[temp])
			} else {
				Devices[UID].LastMessage = _message
				Devices[UID].Name = _wbc.CTs.CT[temp].Name
				Devices[UID].Level = (_wbs.Tmps.Tmp[temp].Value / 16)
				passMessage("existingtempupdated", *Devices[UID])
				myLog.Infof("        **** Updating Temperature Device for ", UID, _wbc.CTs.CT[temp])
			}

		} else {
			myLog.Infof("        **** Excluding Temperature Device for ", UID)

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

func SetLightLevel(devID string, level float64) (bool, error) {

	var command string

	// update the record for new levels
	Devices[devID].Level = float64(level)
	//var statebit string
	if Devices[devID].Level == 0 {
		Devices[devID].State = false
	} else {
		Devices[devID].State = true
	}

	// create and send the command

	command = "http://" + Devices[devID].IP.String() + "/hid.spi?com=%3A&com=AA" + strconv.Itoa(Devices[devID].Channel) + "%3B" + strconv.FormatFloat((Devices[devID].Level*100), 'f', 0, 64) + "&com=%3A"

	// http://192.168.1.249/hid.spi?com=%3A&com=AA0%3B85&com=%3A

	myLog.Debugf("++++++++++++ in SetLevel for LIGHT with %s ++++++++++++\n", command)
	success, err := sendCommand(command, devID)

	passMessage("lightset:"+strconv.FormatFloat(Devices[devID].Level, 'f', 6, 64), *Devices[devID])
	command = ""
	return success, err

}

// SetState sets the state of a device
func SetState(devID string, state bool) (bool, error) {

	var command, _wbstate string
	var _level float64
	var _success bool
	var _err error

	_err = nil
	_success = true

	// update the record for new levels
	Devices[devID].State = state

	myLog.Debugf("++++++++++++ in SetState for State on %s with %s ++++++++++++\n", devID, command)

	// Convert state to the webbrick, and override the level if it's a light
	if state {
		_wbstate = "N" // On
		if Devices[devID].Level == 0 {
			_level = 0.95
		} else {
			_level = Devices[devID].Level
		}
	} else {
		_wbstate = "F" // Off
		_level = 0
	}

	// if the state is on a light device then check and set the light level
	if Devices[devID].Type == LIGHT {
		// create and send the command
		command = "http://" + Devices[devID].IP.String() + "/hid.spi?com=%3A&com=AA" + strconv.Itoa(Devices[devID].Channel) + "%3B" + strconv.FormatFloat((_level*100), 'f', 0, 64) + "&com=%3A"

		myLog.Debugf("++++++++++++ in SetState for Level on %s with %s ++++++++++++\n", devID, command)
		success, err := sendCommand(command, devID)
		if err != nil {
			myLog.Errorf("Error setting light level in state for ", command, "\n", err)
			_err = err
		} else {
			_success = success
		}
		passMessage("stateset:"+strconv.FormatFloat(Devices[devID].Level, 'f', 6, 64), *Devices[devID])
	} else {

		// create and send the command
		command = "http://" + Devices[devID].IP.String() + "/hid.spi?com=%3A&com=DO" + strconv.Itoa(Devices[devID].Channel) + "%3B" + _wbstate + "&com=%3A"

		myLog.Debugf("++++++++++++ in SetState for State on %s with %s ++++++++++++\n", devID, command)
		success, err := sendCommand(command, devID)
		if err != nil {
			myLog.Errorf("Error setting light level in state for ", command)
			_err = err
		} else {
			_success = success
		}
		passMessage("stateset:"+strconv.FormatFloat(Devices[devID].Level, 'f', 6, 64), *Devices[devID])
	}

	command = ""
	return _success, _err

}

// SetState sets the state of a socket, given its MAC address
func PushButton(devID string) (bool, error) {

	var command string

	// create and send the command
	command = "http://" + Devices[devID].IP.String() + "/hid.spi?com=%3A&com=DI" + strconv.Itoa(Devices[devID].Channel) + "&com=%3A"

	myLog.Debugf("Push button ", command)

	myLog.Debugf("++++++++++++ in PushButton and going to send %s ++++++++++++\n", command)
	success, err := sendCommand(command, devID)

	passMessage("button", *Devices[devID])
	command = ""
	return success, err

}

////////////////////////////////////////////////
//
// Helpers
//
////////////////////////////////////////////////

// GetState gets the state of a device, given its ID
func GetState(devID string) bool {
	return Devices[devID].State
}

// GetLevel gets the level of a device, given its ID
func GetLevel(devID string) float64 {
	return Devices[devID].Level
}

// GetLevel gets the level of a device, given its ID
func GetLastMessage(devID string) string {
	return Devices[devID].LastMessage
}

// ==================
// Internal functions
// ==================

// handleMessage parses a message found by CheckForMessages
func handleMessage(buf []byte, addr *net.UDPAddr) (bool, error) {

	var _tmpValue = 0

	//Strip out the information sent from the brick
	resp := new(WebBrickMsg)
	resp.Addr = addr.IP.String()
	//resp.PacketSource = strings.ToUpper(resp.PacketSource)

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
				resp.SourceChannel = int(element)
			}
		case 5:
			switch resp.PacketSource {
			case "ST": // Handle time message differently
				resp.Minute = strconv.Itoa(int(element))
			default:
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
			switch strings.ToUpper(resp.PacketSource) {
			case "AO", "AI", "CT":
				resp.Value = strconv.Itoa(int(element))
				_tmpValue = int(element)
			default:
			}
		case 12:
			switch strings.ToUpper(resp.PacketSource) {
			case "CT":
				resp.Value = strconv.Itoa(int(element) + _tmpValue)
			default:
			}
		default:

		}

		// index is the index where we are
		// element is the element from someSlice for where we are

		_checkSource := strings.ToUpper(resp.PacketSource)

		if DEBUG {
			fmt.Println(" Handler for " + resp.PacketSource + "(" + _checkSource + ")")
			fmt.Println(index)
			fmt.Printf(" : ")
			fmt.Println((reflect.TypeOf(element)))
			fmt.Printf(" : ")
			fmt.Println((element))
			fmt.Printf(" : ")
			fmt.Println(string(element))
			fmt.Printf("\n")
		}

		if index > 3 && _checkSource != "ST" && _checkSource != "CT" && _checkSource != "AO" && _checkSource != "DO" && _checkSource != "TD" {
			myLog.Errorf("Unknown Device type found : ", resp.PacketSource, _checkSource)

			if DEBUG {
				fmt.Println("Missing Handler for " + resp.PacketSource + "(" + _checkSource + ")")
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

	UID := strconv.Itoa(resp.FromNodeNo) + "::" + strings.ToUpper(resp.PacketSource) + "::" + strconv.Itoa(resp.SourceChannel)

	myLog.Infof(UID + " seen ")

	switch strings.ToUpper(resp.PacketSource) {
	case "ST": // Timestamp

		_message := "Seen at " + resp.Hour + ":" + resp.Minute + ":" + resp.Second

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", resp.FromNodeNo, HEARTBEAT, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
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
			Devices[UID] = &Device{deviceCount, UID, "", resp.FromNodeNo, PIR, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
			passMessage("newtriggerfound", *Devices[UID])
		} else {
			Devices[UID].LastMessage = _message
			passMessage("existingtriggerupdated", *Devices[UID])
		}

	case "CT": // Temp (?)

		// Calculate the local values
		_value, _ := strconv.ParseFloat(resp.Value, 64)
		_message := "Temp on " + strconv.Itoa(resp.SourceChannel) + " at " + strconv.FormatFloat((_value), 'f', 2, 64)

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", resp.FromNodeNo, TEMP, resp.SourceChannel, addr.IP, true, false, false, _value / 16, _message}
			passMessage("newtempfound", *Devices[UID])
		} else {
			Devices[UID].LastMessage = _message
			Devices[UID].Level = (_value / 16)
			passMessage("existingtempupdated", *Devices[UID])
		}

	case "TD": // Button (?) Check is this includes PIR as well

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if !PIRS[UID] {
			_message := "Button pressed on " + strconv.Itoa(resp.SourceChannel)

			if ok == false { // we haven't got this in our Devices array
				deviceCount++
				Devices[UID] = &Device{deviceCount, UID, "", resp.FromNodeNo, BUTTON, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
				passMessage("newbuttonfound", *Devices[UID])
			} else {
				Devices[UID].LastMessage = _message
				Devices[UID].State = true
				passMessage("existingbuttonupdated", *Devices[UID])
			}

		} else {

			_message := "PIR actioned on " + strconv.Itoa(resp.SourceChannel)

			if ok == false { // we haven't got this in our Devices array
				deviceCount++
				Devices[UID] = &Device{deviceCount, UID, "", resp.FromNodeNo, PIR, resp.SourceChannel, addr.IP, true, false, false, 0, _message}
				passMessage("newpirfound", *Devices[UID])
			} else {
				Devices[UID].LastMessage = _message
				Devices[UID].State = true
				passMessage("existingpirtriggered", *Devices[UID])
			}

		}

	case "AO": // Light Dimmer Device

		// Calculate the private values for the message
		var _state bool
		_message := "Light at level " + resp.Value
		//_value, _ := strconv.Atoi(resp.Value)
		_value, _ := strconv.ParseFloat(resp.Value, 64)
		if _value > 0 {
			_state = true
		} else {
			_state = false
		}

		// Check to see if we've already got macAdd in our array
		_, ok := Devices[UID]

		if ok == false { // we haven't got this in our Devices array
			deviceCount++
			Devices[UID] = &Device{deviceCount, UID, "", resp.FromNodeNo, LIGHT, resp.SourceChannel, addr.IP, true, false, _state, _value, _message}
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
	myLog.Infof("**** Body Message ****", body)

	// Looks good
	return true, nil

}
