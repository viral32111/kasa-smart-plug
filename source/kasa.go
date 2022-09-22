package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"strings"
	"time"
)

const KASA_INITIAL_KEY int = 171

/*
https://www.softscheck.com/en/reverse-engineering-tp-link-hs110/
	https://github.com/softScheck/tplink-smartplug/blob/master/tplink-smarthome-commands.txt

https://www.bencode.net/papers/2021-simmonds-radiosec-tplink-kp115-teardown.pdf
https://github.com/SimonWilkinson/python-kasa

system -> reset -> delay = X
	Factory reset the plug after X seconds

netif -> get_scaninfo -> refresh = X
	Get a list of available wireless networks
*/

type QueryResponse struct {
	System struct {
		Info struct {
			SoftwareVersion string `json:"sw_ver"`
			HardwareVersion string `json:"hw_ver"`
			Model string `json:"model"`
			DeviceIdentifier string `json:"deviceId"`
			OEMIdentifier string `json:"oemId"`
			HardwareIdentifier string `json:"hwId"`
			SignalStrength int `json:"rssi"`
			Latitude int `json:"latitude_i"`
			Longitude int `json:"longitude_i"`
			Alias string `json:"alias"`
			Status string `json:"status"`
			Source string `json:"obd_src"`
			Type string `json:"mic_type"`
			Features string `json:"feature"`
			MACAddress string `json:"mac"`
			Updating int `json:"updating"`
			LEDOff int `json:"led_off"`
			RelayState int `json:"relay_state"`
			UptimeSeconds int `json:"on_time"`
			IconHash string `json:"icon_hash"`
			DeviceName string `json:"dev_name"`
			ActiveMode string `json:"active_mode"`
			NextAction struct {
				Type int `json:"type"`
				Identifier string `json:"id"`
				ScheduledSeconds int `json:"schd_sec"`
				Action int `json:"action"`
			} `json:"next_action"`
			NTCState int `json:"ntc_state"`
			ErrorCode int `json:"err_code"`
		} `json:"get_sysinfo"`

		RelayState struct {
			ErrorCode int `json:"err_code"`
		} `json:"set_relay_state"`

		LEDOff struct {
			ErrorCode int `json:"err_code"`
		} `json:"set_led_off"`
	} `json:"system"`

	Time struct {
		Now struct {
			Year int `json:"year"`
			Month int `json:"month"`
			Day int `json:"mday"`
			Hour int `json:"hour"`
			Minute int `json:"min"`
			Second int `json:"sec"`
			ErrorCode int `json:"err_code"`
		} `json:"get_time"`

		Zone struct {
			Index int `json:"index"`
			ErrorCode int `json:"err_code"`
		} `json:"get_timezone"`
	} `json:"time"`

	EnergyMeter struct {
		Now struct {
			Amperage int `json:"current_ma"` // milliamps
			Voltage int `json:"voltage_mv"` // millivolts
			Wattage int `json:"power_mw"` // milliwatts
			Total int `json:"total_wh"` // watthours
			ErrorCode int `json:"err_code"`
		} `json:"get_realtime"`

		Daily struct {
			Days []struct {
				Year int `json:"year"`
				Month int `json:"month"`
				Day int `json:"day"`
				Total int `json:"energy_wh"` // watthours
			} `json:"day_list"`
			ErrorCode int `json:"err_code"`
		} `json:"get_daystat"`

		Monthly struct {
			Months []struct {
				Year int `json:"year"`
				Month int `json:"month"`
				Total int `json:"energy_wh"` // watthours
			} `json:"month_list"`
			ErrorCode int `json:"err_code"`
		} `json:"get_monthstat"`
	} `json:"emeter"`
}

type KasaSmartPlug struct {
	/******************** Private *******************/
	plugConnection net.Conn

	/******************** Public *******************/
	Alias string
	Icon string
	PowerState bool
	LightState bool
	Uptime int

	DeviceName string
	DeviceModel string
	DeviceIdentifier string
	DeviceFeatures []string
	HardwareVersion string
	HardwareIdentifier string
	OEMIdentifier string

	Status string
	FirmwareUpdating bool
	FirmwareVersion string

	SignalStrength int
	MACAddress string

	Latitude float64
	Longitude float64

	// TO-DO: Work out what these three are...
	Source string
	Type string
	NTCState int

	Action struct {
		Name string
		Type int
		Identifier string
		ScheduledSeconds int
		Action int
	}

	ErrorCode int

	Time time.Time

	Energy struct {
		Amperage float64
		Voltage float64
		Wattage float64
		Total int
	}

	// TO-DO: Historical energy usage structure
}

func Connect( host string, port int ) KasaSmartPlug {
	connection, _ := net.Dial( "tcp", fmt.Sprintf( "%s:%d", host, port ) ) // TO-DO: Use DialTimeout() instead

	smartPlug := KasaSmartPlug {
		plugConnection: connection,
	}

	smartPlug.Update()

	return smartPlug
}

func ( smartPlug KasaSmartPlug ) Disconnect() {
	smartPlug.plugConnection.Close()
}

func ( smartPlug KasaSmartPlug ) encryptQuery( data []byte ) []byte {
	encryptedData := make( []byte, len( data ) )
	key := KASA_INITIAL_KEY

	for index := 0; index < len( data ); index++ {
		key = key ^ int( data[ index ] )
		encryptedData[ index ] = byte( key )
	}

	return encryptedData
}

func ( smartPlug KasaSmartPlug ) decryptQuery( data []byte ) []byte {
	decryptedData := make( []byte, len( data ) )
	key := KASA_INITIAL_KEY

	for index := 0; index < len( data ); index++ {
		encryptedCharacter := int( data[ index ] )
		decryptedData[ index ] = byte( key ^ encryptedCharacter )
		key = encryptedCharacter
	}

	return decryptedData
}

func ( smartPlug KasaSmartPlug ) sendQuery( target string, command string, data map[string]int ) QueryResponse {
	/******************** Construct *******************/
	jsonPayload, _ := json.Marshal( map[string]map[string]map[string]int {
		target: {
			command: data,
		},
	} )

	/******************** Send *******************/
	var queryBuffer bytes.Buffer
	binary.Write( &queryBuffer, binary.BigEndian, uint32( len( jsonPayload ) ) )
	queryBuffer.Write( smartPlug.encryptQuery( []byte( jsonPayload ) ) )

	smartPlug.plugConnection.Write( queryBuffer.Bytes() )

	/******************** Receive *******************/
	connectionReader := bufio.NewReader( smartPlug.plugConnection )
	
	responseLengthBytes := make( []byte, 4 )
	binary.Read( connectionReader, binary.BigEndian, responseLengthBytes )

	jsonResponseBytes := make( []byte, binary.BigEndian.Uint32( responseLengthBytes ) )
	binary.Read( connectionReader, binary.BigEndian, jsonResponseBytes )

	// This is for debugging responses
	//fmt.Println( string( smartPlug.decryptQuery( jsonResponseBytes ) ) )

	/******************** Parse *******************/
	var queryResponse QueryResponse
	json.Unmarshal( smartPlug.decryptQuery( jsonResponseBytes ), &queryResponse )

	return queryResponse
}

func ( smartPlug *KasaSmartPlug ) Update() bool {
	response := smartPlug.sendQuery( "system", "get_sysinfo", map[string]int{} )

	smartPlug.Alias = response.System.Info.Alias
	smartPlug.Icon = response.System.Info.IconHash
	smartPlug.PowerState = ( response.System.Info.RelayState != 0 )
	smartPlug.LightState = ( response.System.Info.LEDOff == 0 )
	smartPlug.Uptime = response.System.Info.UptimeSeconds

	smartPlug.DeviceName = response.System.Info.DeviceName
	smartPlug.DeviceModel = response.System.Info.Model
	smartPlug.DeviceIdentifier = response.System.Info.DeviceIdentifier
	smartPlug.DeviceFeatures = strings.Split( response.System.Info.Features, ":" )
	smartPlug.HardwareVersion = response.System.Info.HardwareVersion
	smartPlug.HardwareIdentifier = response.System.Info.HardwareIdentifier
	smartPlug.OEMIdentifier = response.System.Info.OEMIdentifier

	smartPlug.Status = response.System.Info.Status
	smartPlug.FirmwareUpdating = ( response.System.Info.Updating != 0 )
	smartPlug.FirmwareVersion = response.System.Info.SoftwareVersion

	smartPlug.SignalStrength = response.System.Info.SignalStrength
	smartPlug.MACAddress = response.System.Info.MACAddress

	smartPlug.Latitude = float64( response.System.Info.Latitude ) / 10000.0
	smartPlug.Longitude = float64( response.System.Info.Longitude ) / 10000.0

	smartPlug.Source = response.System.Info.Source
	smartPlug.Type = response.System.Info.Type
	smartPlug.NTCState = response.System.Info.NTCState

	smartPlug.Action.Name = response.System.Info.ActiveMode
	smartPlug.Action.Type = response.System.Info.NextAction.Type
	smartPlug.Action.Identifier = response.System.Info.NextAction.Identifier
	smartPlug.Action.ScheduledSeconds = response.System.Info.NextAction.ScheduledSeconds
	smartPlug.Action.Action = response.System.Info.NextAction.Action

	smartPlug.ErrorCode = response.System.Info.ErrorCode

	smartPlug.updateTime()

	return ( smartPlug.ErrorCode == 0 )
}

func ( smartPlug *KasaSmartPlug ) updateTime() bool {
	timeResponse := smartPlug.sendQuery( "time", "get_time", map[string]int {} )
	smartPlug.ErrorCode = timeResponse.Time.Now.ErrorCode

	zoneResponse := smartPlug.sendQuery( "time", "get_timezone", map[string]int {} )
	smartPlug.ErrorCode = zoneResponse.Time.Zone.ErrorCode

	zoneOffset := 0 // 39 is Europe/London?

	smartPlug.Time, _ = time.Parse( "2006-01-02 15:04:05 -0700", fmt.Sprintf( "%04d-%02d-%02d %02d:%02d:%02d -%06d", timeResponse.Time.Now.Year, timeResponse.Time.Now.Month, timeResponse.Time.Now.Day, timeResponse.Time.Now.Hour, timeResponse.Time.Now.Minute, timeResponse.Time.Now.Second, zoneOffset ) )

	return ( smartPlug.ErrorCode == 0 )
}

func ( smartPlug *KasaSmartPlug ) PowerOn() bool {
	smartPlug.Update()

	if smartPlug.PowerState {
		return false
	}

	response := smartPlug.sendQuery( "system", "set_relay_state", map[string]int { "state": 1 } )
	smartPlug.ErrorCode = response.System.RelayState.ErrorCode

	return ( smartPlug.ErrorCode == 0 )
}

func ( smartPlug KasaSmartPlug ) PowerOff() bool {
	smartPlug.Update()

	if !smartPlug.PowerState {
		return false
	}

	response := smartPlug.sendQuery( "system", "set_relay_state", map[string]int { "state": 0 } )
	smartPlug.ErrorCode = response.System.RelayState.ErrorCode

	return ( smartPlug.ErrorCode == 0 )
}

func ( smartPlug KasaSmartPlug ) PowerToggle() bool {
	smartPlug.Update()

	powerState := 0

	if smartPlug.PowerState {
		powerState = 0
	} else if !smartPlug.PowerState {
		powerState = 1
	}

	response := smartPlug.sendQuery( "system", "set_relay_state", map[string]int { "state": powerState } )
	smartPlug.ErrorCode = response.System.RelayState.ErrorCode

	return ( smartPlug.ErrorCode == 0 )
}

func ( smartPlug *KasaSmartPlug ) LightOn() bool {
	smartPlug.Update()

	if smartPlug.LightState {
		return false
	}

	response := smartPlug.sendQuery( "system", "set_led_off", map[string]int { "off": 0 } )
	smartPlug.ErrorCode = response.System.LEDOff.ErrorCode

	return ( smartPlug.ErrorCode == 0 )
}

func ( smartPlug *KasaSmartPlug ) LightOff() bool {
	smartPlug.Update()

	if !smartPlug.LightState {
		return false
	}

	response := smartPlug.sendQuery( "system", "set_led_off", map[string]int { "off": 1 } )
	smartPlug.ErrorCode = response.System.LEDOff.ErrorCode

	return ( smartPlug.ErrorCode == 0 )
}

func ( smartPlug KasaSmartPlug ) LightToggle() bool {
	smartPlug.Update()

	lightState := 0

	if smartPlug.LightState {
		lightState = 1
	} else if !smartPlug.LightState {
		lightState = 0
	}

	response := smartPlug.sendQuery( "system", "set_led_off", map[string]int { "off": lightState } )
	smartPlug.ErrorCode = response.System.LEDOff.ErrorCode

	return ( smartPlug.ErrorCode == 0 )
}

func ( smartPlug KasaSmartPlug ) GetTime() time.Time {
	smartPlug.updateTime()

	return smartPlug.Time
}

func ( smartPlug KasaSmartPlug ) GetPowerTime() time.Time {
	smartPlug.Update()

	if !smartPlug.PowerState {
		return time.Unix( 0, 0 )
	}

	return smartPlug.Time.Add( time.Duration( -smartPlug.Uptime ) * time.Second )
}

func ( smartPlug KasaSmartPlug ) Reboot( delay int ) bool {
	response := smartPlug.sendQuery( "system", "reboot", map[string]int { "delay": int( math.Max( 1.0, float64( delay ) ) ) } )
	smartPlug.ErrorCode = response.System.RelayState.ErrorCode

	return ( smartPlug.ErrorCode == 0 )
}

func ( smartPlug *KasaSmartPlug ) GetEnergyUsage() int {
	response := smartPlug.sendQuery( "emeter", "get_realtime", map[string]int {} )
	smartPlug.ErrorCode = response.System.LEDOff.ErrorCode

	smartPlug.Energy.Amperage = float64( response.EnergyMeter.Now.Amperage ) / 1000.0
	smartPlug.Energy.Voltage = float64( response.EnergyMeter.Now.Voltage ) / 1000.0
	smartPlug.Energy.Wattage = float64( response.EnergyMeter.Now.Wattage ) / 1000.0
	smartPlug.Energy.Total = response.EnergyMeter.Now.Total

	return smartPlug.Energy.Total
}

/*func main() {
	argumentCount := len( os.Args[ 1: ] )

	if ( argumentCount < 1 ) {
		fmt.Printf( "Usage: %s <ip address> [port number]\n", os.Args[ 0 ] )
		os.Exit( 1 )
	}

	portNumber := 9999
	if ( argumentCount >= 2 ) {
		customPortNumber, _ := strconv.ParseInt( os.Args[ 2 ], 10, 64 )
		portNumber = int( customPortNumber )
	}

	smartPlug := Connect( os.Args[ 1 ], portNumber )
	defer smartPlug.Disconnect()

	fmt.Printf( "%s %s '%s' power state is: %t\n", smartPlug.DeviceName, smartPlug.DeviceModel, smartPlug.Alias, smartPlug.PowerState )
	fmt.Printf( "%.2fkWh of energy has been used, and %.2fw of energy is currently being used.\n", float64( smartPlug.GetEnergyUsage() ) / 10000.0, smartPlug.Energy.Wattage )

	//smartPlug.PowerOff()
}*/
