package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"time"
)

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

// Structure for parsing the JSON query responses
type KasaQueryResponse struct {
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

// Structure for holding data about & methods for a smart plug
type KasaSmartPlug struct {

	// The underlying TCP connection
	Connection net.Conn

	// The initial key for encrypting & decrypting data
	InitialKey int

	// Runtime & state
	Alias string
	Icon string
	PowerState bool
	LightState bool
	Uptime int

	// Device information
	DeviceName string
	DeviceModel string
	DeviceIdentifier string
	DeviceFeatures []string
	HardwareVersion string
	HardwareIdentifier string
	OEMIdentifier string

	// Status & firmware
	Status string
	FirmwareUpdating bool
	FirmwareVersion string

	// Network
	SignalStrength int
	MACAddress string

	// Position
	Latitude float64
	Longitude float64

	// TO-DO: Work out what these three are...
	Source string
	Type string
	NTCState int

	// Current action
	Action struct {
		Name string
		Type int
		Identifier string
		ScheduledSeconds int
		Action int
	}

	// Time
	Time time.Time

	// Energy usage
	Energy struct {
		Amperage float64
		Voltage float64
		Wattage float64
		Total int
	}

	// TO-DO: Historical energy usage structure
}

// Opens a connection to a smart plug
func ( smartPlug *KasaSmartPlug ) Connect( address net.IP, port int, timeout int ) ( error ) {

	// Try to open a TCP connection to the smart plug
	connection, connectError := net.DialTimeout( "tcp4", fmt.Sprintf( "%s:%d", address.String(), port ), time.Millisecond * time.Duration( timeout ) )
	if ( connectError != nil ) {
		return connectError
	}

	// Set the connection in the smart plug structure
	smartPlug.Connection = connection

	// Return no error
	return nil

}

// Closes the connection with the smart plug
func ( smartPlug *KasaSmartPlug ) Disconnect() ( error ) {

	// Try to close the connecntion
	closeError := smartPlug.Connection.Close()
	if ( closeError != nil ) {
		return closeError
	}

	// Return no error
	return nil
}

// Encrypts data, usually for sending
func ( smartPlug *KasaSmartPlug ) EncryptData( originalData []byte ) ( []byte ) {
	
	// Create a byte array to hold the encrypted data
	encryptedData := make( []byte, len( originalData ) )

	// The key changes changes with each byte, but the initial key is always the same
	key := smartPlug.InitialKey

	// Update the key, XOR each byte with the current key, then add it to the byte array
	for index := 0; index < len( originalData ); index++ {
		key = key ^ int( originalData[ index ] )
		encryptedData[ index ] = byte( key )
	}

	// Return the byte array containing the encrypted data
	return encryptedData

}

// Decrypts data, usually for receiving
func ( smartPlug *KasaSmartPlug ) DecryptData( encryptedData []byte ) ( []byte ) {

	// Create a byte array to hold the decrypted data
	decryptedData := make( []byte, len( encryptedData ) )

	// The key changes changes with each byte, but the initial key is always the same
	key := smartPlug.InitialKey

	// XOR each byte with the current key, add it to the byte array, then update the key
	for index := 0; index < len( encryptedData ); index++ {
		encryptedCharacter := int( encryptedData[ index ] )
		decryptedData[ index ] = byte( key ^ encryptedCharacter )
		key = encryptedCharacter
	}

	// Return the byte array containing the decrypted data
	return decryptedData

}

// Sends a query to the smart plug
func ( smartPlug *KasaSmartPlug ) SendQuery( targetName string, commandName string, extraData map[string]int ) ( KasaQueryResponse, error ) {

	// Create the JSON payload containing the query
	jsonPayload, encodeError := json.Marshal( map[string]map[string]map[string]int {
		targetName: {
			commandName: extraData,
		},
	} )

	// Fail if there was an error encoding the JSON
	if ( encodeError != nil ) {
		return KasaQueryResponse{}, encodeError
	}

	// Create a binary buffer to hold the encrypted payload
	var queryBuffer bytes.Buffer

	// Write the length of the payload into the buffer
	queryLengthWriteError := binary.Write( &queryBuffer, binary.BigEndian, uint32( len( jsonPayload ) ) )
	if ( queryLengthWriteError != nil ) {
		return KasaQueryResponse{}, queryLengthWriteError
	}

	// Write the encrypted payload into the buffer
	_, queryWriteError := queryBuffer.Write( smartPlug.EncryptData( []byte( jsonPayload ) ) )
	if ( queryWriteError != nil ) {
		return KasaQueryResponse{}, queryWriteError
	}

	// Send the binary buffer to the smart plug
	_, writeError := smartPlug.Connection.Write( queryBuffer.Bytes() )
	if ( writeError != nil ) {
		return KasaQueryResponse{}, writeError
	}

	// Create a reader for reading the response
	connectionReader := bufio.NewReader( smartPlug.Connection )

	// Read the encrypted response payload length (32-bit integer)
	responseLengthBytes := make( []byte, 4 )
	responseLengthReadError := binary.Read( connectionReader, binary.BigEndian, responseLengthBytes )
	if ( responseLengthReadError != nil ) {
		return KasaQueryResponse{}, responseLengthReadError
	}

	// Read the encrypted response payload
	responseBytes := make( []byte, binary.BigEndian.Uint32( responseLengthBytes ) )
	responseReadError := binary.Read( connectionReader, binary.BigEndian, responseBytes )
	if ( responseReadError != nil ) {
		return KasaQueryResponse{}, responseReadError
	}

	// Decrypt the response payload and parse it as JSON into the response structure
	var queryResponse KasaQueryResponse
	decodeError := json.Unmarshal( smartPlug.DecryptData( responseBytes ), &queryResponse )
	if ( decodeError != nil ) {
		return KasaQueryResponse{}, decodeError
	}

	// Return the response
	return queryResponse, nil

}

// Updates all the properties with the latest data
func ( smartPlug *KasaSmartPlug ) UpdateProperties() ( error ) {

	// Fetch the system information
	queryResponse, sendError := smartPlug.SendQuery( "system", "get_sysinfo", map[string]int{} )
	if ( sendError != nil ) {
		return sendError
	}

	// Update runtime & state properties
	smartPlug.Alias = queryResponse.System.Info.Alias
	smartPlug.Icon = queryResponse.System.Info.IconHash
	smartPlug.PowerState = ( queryResponse.System.Info.RelayState != 0 )
	smartPlug.LightState = ( queryResponse.System.Info.LEDOff == 0 )
	smartPlug.Uptime = queryResponse.System.Info.UptimeSeconds

	// Update device information properties
	smartPlug.DeviceName = queryResponse.System.Info.DeviceName
	smartPlug.DeviceModel = queryResponse.System.Info.Model
	smartPlug.DeviceIdentifier = queryResponse.System.Info.DeviceIdentifier
	smartPlug.DeviceFeatures = strings.Split( queryResponse.System.Info.Features, ":" )
	smartPlug.HardwareVersion = queryResponse.System.Info.HardwareVersion
	smartPlug.HardwareIdentifier = queryResponse.System.Info.HardwareIdentifier
	smartPlug.OEMIdentifier = queryResponse.System.Info.OEMIdentifier

	// Update status & firmware properties
	smartPlug.Status = queryResponse.System.Info.Status
	smartPlug.FirmwareUpdating = ( queryResponse.System.Info.Updating != 0 )
	smartPlug.FirmwareVersion = queryResponse.System.Info.SoftwareVersion

	// Update network properties
	smartPlug.SignalStrength = queryResponse.System.Info.SignalStrength
	smartPlug.MACAddress = queryResponse.System.Info.MACAddress

	// Update position properties
	smartPlug.Latitude = float64( queryResponse.System.Info.Latitude ) / 10000.0
	smartPlug.Longitude = float64( queryResponse.System.Info.Longitude ) / 10000.0

	// Update ???? properties
	smartPlug.Source = queryResponse.System.Info.Source
	smartPlug.Type = queryResponse.System.Info.Type
	smartPlug.NTCState = queryResponse.System.Info.NTCState

	// Update current action properties
	smartPlug.Action.Name = queryResponse.System.Info.ActiveMode
	smartPlug.Action.Type = queryResponse.System.Info.NextAction.Type
	smartPlug.Action.Identifier = queryResponse.System.Info.NextAction.Identifier
	smartPlug.Action.ScheduledSeconds = queryResponse.System.Info.NextAction.ScheduledSeconds
	smartPlug.Action.Action = queryResponse.System.Info.NextAction.Action

	// Update the time-related properties
	updateTimeError := smartPlug.UpdateTimeProperties()
	if ( updateTimeError != nil ) {
		return updateTimeError
	}

	// Fail if there is an error set
	if ( queryResponse.System.Info.ErrorCode != 0 ) {
		return fmt.Errorf( "%d", queryResponse.System.Info.ErrorCode )
	}

	// Update the energy usage properties
	updateEnergyUsageError := smartPlug.UpdateEnergyUsageProperties()
	if ( updateEnergyUsageError != nil ) {
		return updateEnergyUsageError
	}

	// Return no error if we got this far
	return nil

}

// Updates the time-related properties
func ( smartPlug *KasaSmartPlug ) UpdateTimeProperties() ( error ) {

	// Fetch the current time
	timeResponse, timeQueryError := smartPlug.SendQuery( "time", "get_time", map[string]int {} )
	if ( timeQueryError != nil ) {
		return timeQueryError
	}

	// Fail if there is an error set
	if ( timeResponse.Time.Now.ErrorCode != 0 ) {
		return fmt.Errorf( "%d", timeResponse.Time.Now.ErrorCode )
	}

	// Fetch the timezone
	zoneResponse, zoneQueryError := smartPlug.SendQuery( "time", "get_timezone", map[string]int {} )
	if ( zoneQueryError != nil ) {
		return zoneQueryError
	}

	// Fail if there is an error set
	if ( zoneResponse.Time.Zone.ErrorCode != 0 ) {
		return fmt.Errorf( "%d", timeResponse.Time.Zone.ErrorCode )
	}

	// TODO: Get the timezone offset from the timezone response
	zoneOffset := 0 // 39 is Europe/London?

	// Parse the date & time from the response
	parsedTime, parseError := time.Parse( "2006-01-02 15:04:05 -070000", fmt.Sprintf( "%04d-%02d-%02d %02d:%02d:%02d -%06d", timeResponse.Time.Now.Year, timeResponse.Time.Now.Month, timeResponse.Time.Now.Day, timeResponse.Time.Now.Hour, timeResponse.Time.Now.Minute, timeResponse.Time.Now.Second, zoneOffset ) )
	if ( parseError != nil ) {
		return parseError
	}

	// Update the properties
	smartPlug.Time = parsedTime

	// Return no error if we got this far
	return nil

}

// Updates the energy usage properties
func ( smartPlug *KasaSmartPlug ) UpdateEnergyUsageProperties() ( error ) {

	// Send the energy usage command
	queryResponse, queryError := smartPlug.SendQuery( "emeter", "get_realtime", map[string]int {} )
	if ( queryError != nil ) {
		return queryError
	}

	// Update the properties on the structure
	smartPlug.Energy.Amperage = float64( queryResponse.EnergyMeter.Now.Amperage ) / 1000.0
	smartPlug.Energy.Voltage = float64( queryResponse.EnergyMeter.Now.Voltage ) / 1000.0
	smartPlug.Energy.Wattage = float64( queryResponse.EnergyMeter.Now.Wattage ) / 1000.0
	smartPlug.Energy.Total = queryResponse.EnergyMeter.Now.Total

	// Return no error
	return nil

}

// Switches on power
func ( smartPlug *KasaSmartPlug ) PowerOn() ( error ) {

	// Update data
	updateError := smartPlug.UpdateProperties()
	if ( updateError != nil ) {
		return updateError
	}

	// Fail if the plug is already switched on
	if ( smartPlug.PowerState ) {
		return errors.New( "smart plug is already powered on" )
	}

	// Send the power on command
	queryResponse, queryError := smartPlug.SendQuery( "system", "set_relay_state", map[string]int { "state": 1 } )
	if ( queryError != nil ) {
		return queryError
	}

	// Fail if there is an error set
	if ( queryResponse.System.RelayState.ErrorCode != 0 ) {
		return fmt.Errorf( "%d", queryResponse.System.RelayState.ErrorCode )
	}

	// Return no error
	return nil

}

// Switches off power
func ( smartPlug *KasaSmartPlug ) PowerOff() ( error ) {
	
	// Update data
	updateError := smartPlug.UpdateProperties()
	if ( updateError != nil ) {
		return updateError
	}

	// Fail if the plug is already switched off
	if ( !smartPlug.PowerState ) {
		return errors.New( "smart plug is already powered off" )
	}

	// Send the power off command
	queryResponse, queryError := smartPlug.SendQuery( "system", "set_relay_state", map[string]int { "state": 0 } )
	if ( queryError != nil ) {
		return queryError
	}

	// Fail if there is an error set
	if ( queryResponse.System.RelayState.ErrorCode != 0 ) {
		return fmt.Errorf( "%d", queryResponse.System.RelayState.ErrorCode )
	}

	// Return no error
	return nil

}

// Toggle power
func ( smartPlug KasaSmartPlug ) PowerToggle() ( error ) {

	// Update data
	updateError := smartPlug.UpdateProperties()
	if ( updateError != nil ) {
		return updateError
	}

	// Get the current power state
	powerState := 0
	if ( smartPlug.PowerState ) {
		powerState = 0
	} else if ( !smartPlug.PowerState ) {
		powerState = 1
	}

	// Send the power command
	queryResponse, queryError := smartPlug.SendQuery( "system", "set_relay_state", map[string]int { "state": powerState } )
	if ( queryError != nil ) {
		return queryError
	}

	// Fail if there is an error set
	if ( queryResponse.System.RelayState.ErrorCode != 0 ) {
		return fmt.Errorf( "%d", queryResponse.System.RelayState.ErrorCode )
	}

	// Return no error
	return nil

}

// Switch on the power indicator light
func ( smartPlug *KasaSmartPlug ) LightOn() ( error ) {
	
	// Update data
	updateError := smartPlug.UpdateProperties()
	if ( updateError != nil ) {
		return updateError
	}

	// Fail if the light is already on
	if ( smartPlug.LightState ) {
		return errors.New( "smart plug light is already on" )
	}

	// Send the light on command
	queryResponse, queryError := smartPlug.SendQuery( "system", "set_led_off", map[string]int { "off": 0 } )
	if ( queryError != nil ) {
		return queryError
	}

	// Fail if there is an error set
	if ( queryResponse.System.LEDOff.ErrorCode != 0 ) {
		return fmt.Errorf( "%d", queryResponse.System.LEDOff.ErrorCode )
	}

	// Return no error
	return nil

}

// Switch off the power indicator light
func ( smartPlug *KasaSmartPlug ) LightOff() ( error ) {

	// Update data
	updateError := smartPlug.UpdateProperties()
	if ( updateError != nil ) {
		return updateError
	}

	// Fail if the light is already off
	if ( !smartPlug.LightState ) {
		return errors.New( "smart plug light is already off" )
	}

	// Send the light off command
	queryResponse, queryError := smartPlug.SendQuery( "system", "set_led_off", map[string]int { "off": 1 } )
	if ( queryError != nil ) {
		return queryError
	}

	// Fail if there is an error set
	if ( queryResponse.System.LEDOff.ErrorCode != 0 ) {
		return fmt.Errorf( "%d", queryResponse.System.LEDOff.ErrorCode )
	}

	// Return no error
	return nil

}

// Toggle the power indicator light
func ( smartPlug KasaSmartPlug ) LightToggle() ( error ) {

	// Update data
	updateError := smartPlug.UpdateProperties()
	if ( updateError != nil ) {
		return updateError
	}

	// Get the current light state
	lightState := 0
	if ( smartPlug.LightState ) {
		lightState = 1
	} else if ( !smartPlug.LightState ) {
		lightState = 0
	}

	// Send the light command
	queryResponse, queryError := smartPlug.SendQuery( "system", "set_led_off", map[string]int { "off": lightState } )
	if ( queryError != nil ) {
		return queryError
	}

	// Fail if there is an error set
	if ( queryResponse.System.LEDOff.ErrorCode != 0 ) {
		return fmt.Errorf( "%d", queryResponse.System.LEDOff.ErrorCode )
	}

	// Return no error
	return nil

}

// Get the current time
func ( smartPlug *KasaSmartPlug ) GetTime() ( time.Time, error ) {

	// Update time-related data
	updateError := smartPlug.UpdateTimeProperties()
	if ( updateError != nil ) {
		return time.Unix( 0, 0 ), updateError
	}

	// Return the time
	return smartPlug.Time, nil

}

// Get the power-on time
func ( smartPlug *KasaSmartPlug ) GetPowerTime() ( time.Time, error ) {
	
	// Update data
	updateError := smartPlug.UpdateProperties()
	if ( updateError != nil ) {
		return time.Unix( 0, 0 ), updateError
	}

	// Return zero if the plug is not powered on
	if ( !smartPlug.PowerState ) {
		return time.Unix( 0, 0 ), errors.New( "smart plug is not powered on" )
	}

	// Return the power-on time
	return smartPlug.Time.Add( time.Duration( -smartPlug.Uptime ) * time.Second ), nil

}

// Restart
func ( smartPlug *KasaSmartPlug ) Reboot( delay int ) ( error ) {

	// Send the restart command
	_, queryError := smartPlug.SendQuery( "system", "reboot", map[string]int { "delay": int( math.Max( 1.0, float64( delay ) ) ) } )
	if ( queryError != nil ) {
		return queryError
	}

	// Return no error
	return nil

}
