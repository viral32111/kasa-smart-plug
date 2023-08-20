package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
)

// Metadata
const (
	PROJECT_NAME = "Kasa Smart Plug"
	PROJECT_VERSION = "2.0.1"

	AUTHOR_NAME = "viral32111"
	AUTHOR_WEBSITE = "https://viral32111.com"
)

/*
kasa-smart-plug
	[-h/--help]
		Show this help message and exit.

	[-a/--plug-address <string>]
		The IP address of the smart plug (e.g., 192.168.0.5).
		Scans the local network for a smart plug if not given.
	[-p/--plug-port <number (def. 9999)>]
		The port number of smart plug's API.
	[-k/--initial-key <number (def. 171)>]
		The starting key for XOR encryption & decryption.
		Only change if you know what you are doing!

	[--api-address <string (def. '127.0.0.1')>]
		The IP address to listen on for the HTTP API.
	[--api-port <number (def. 3000)>]
		The port number to listen on for the HTTP API.
		Can serve on the same port as the Prometheus metrics exporter (--metrics-port).
		Set to 0 to disable the HTTP API.
	[--api-path <string (def. '/api')>]
		The HTTP base path of the API routes.
	[-t/--api-tokens <strings>]
		A comma-separated list of API tokens to use for authentication.
		The API is disabled if this is not given.
		A reverse proxy with TLS should be used if this is given, as the tokens are sent in plain-text as HTTP headers and/or URL parameters.
	[-f/--api-tokens-file <string (def. 'api-tokens.txt')>]
		The path to a file containing a new-line separated list of API tokens to use for authentication.
		Each API token may be prefixed with a name followed by a colon, to use in logging instead of the token. This is recommended!
		The API is disabled if the file is empty or does not exist, and --api-tokens is not given.
	[--disable-api-streaming]
		Disables real-time streaming of smart plug updates over WebSocket (/stream/ws) and Server-Sent Events (/stream/sse).
		Requires the Prometheus metrics exporter to be enabled, as it relies upon the periodic metrics collection (--metrics-interval). 
	[--disable-api-documentation]
		Disables the API documentation HTML page at /docs.
		Disables the redirect from / to /docs too.
		It is also available online at https://viral32111.github.io/kasa-smart-plug.
	[--api-documentation-page <string>]
		Override the built-in API documentation HTML page.
	[--disable-api-authentication]
		Disables the API authentication requirements, allowing unrestricted access. This is NOT recommended!
		The --api-tokens & --api-tokens-file flags are ignored if this is given, thus enabling the API.
	[--disable-api-logging]
		Disables logging of API requests/responses to the console.
	[--api-log-file <string (def. 'api.log')>]
		The path to a file to log API requests/responses to.
		Leave blank or set to /dev/null to disable logging to file.

	[--rpc-address <string (def. '127.0.0.1'>]
		The IP address to listen on for the gRPC API.
	[--rpc-port <number (def. 4000)>]
		The port number to listen on for the gRPC API.
		Cannot serve on the same port as the HTTP JSON API (--api-port) or Prometheus metrics exporter (--metrics-port).
		Set to 0 to disable the gRPC API.
	[--rpc-password <string>]
		The password for gRPC authentication.
		The gRPC API is disabled if this is not given.
	[--rpc-tls-certificate <string>]
		The path to a file containing the TLS certificate for the gRPC API.
		Only the first certificate will be used if the file is a chain of certificates.
	[--rpc-tls-key <string>]
		The path to a file containing the TLS private key for the gRPC API.
	[--disable-rpc-authentication]
		Disables the gRPC authentication requirements, allowing unrestricted access. This is NOT recommended!
		The --rpc-password flag is ignored if this is given, thus enabling the gRPC API.
	[--disable-rpc-logging]
		Disables logging of gRPC requests/responses to the console.
	[--rpc-log-file <string (def. 'rpc.log')>]
		The path to a file to log gRPC requests/responses to.
		Leave blank or set to /dev/null to disable logging to file.

	[--metrics-address <string (def. '127.0.0.1')>]
		The IP address to listen on for the HTTP Prometheus metrics exporter.
	[--metrics-port <number (def. 5000)>]
		The port number to listen on for the HTTP Prometheus metrics exporter.
		Can serve on the same port as the JSON API (--api-port).
		Set to 0 to disable the metrics exporter.
	[--metrics-path <string (def. '/metrics')>]
		The HTTP path to the metrics page.
		If serving on the same port as the JSON API, ensure this is not the same as an API route.
	[-u/--metrics-authentication <string>]
		Colon separated username & password for HTTP basic authentication.
		Authentication is disabled if this is not given, allowing unrestricted access.
	[-i/--metrics-interval <number (def. 15)>]
		The time in seconds to wait between collecting metrics.
	[--disable-metrics-logging]
		Disables logging of metrics collection and HTTP requests/responses to the console.
	[--metrics-log-file <string (def. 'metrics.log')>]
		The path to a file to log metrics collection and HTTP requests/responses to.
		Leave blank or set to /dev/null to disable logging to file.

	[-f/--format <human|json (def. 'human')>]
		The output format for commands. Use JSON for machine-readable.

	[command] [arguments...]
		Do not give any commands to act as a daemon, useful for exporting metrics & serving requests from the JSON API.

Commands:
	info
		Returns information about the smart plug.
	usage [now|total|average] [7d|30d]
		Returns the energy usage reported by the smart plug.
	power [on|off]
		Turns the smart plug on or off.
	light [on|off]
		Turns the smart plug's light on or off.

Kasa Smart Plug v2.0.0, by viral32111 (https://viral32111.com).
https://github.com/viral32111/kasa-smart-plug

Copyright (C) 2022-2023 viral32111, under GNU AGPL v3.
*/

/*
kasa [-h/--help] [-a/--address <IPv4 address>] [-p/--port <number (def. 9999)>] [-k/--initial-key <number (def. 171)>] [-f/--format <human|json (def. human)>] [--metrics-address <IPv4 address (def. 127.0.0.1)>] [--metrics-port <number (def. 5000)>] [--metrics-path <string (def. /metrics)>] [-i/--metrics-interval <seconds (def. 15)>] [command] [argument, ...]

kasa -a 192.168.0.5
kasa -a 192.168.0.5 info
kasa -a 192.168.0.5 -f json uptime
kasa --address 192.168.0.5 --port 9999 usage
kasa -a 192.168.0.5 -p 9999 power on
kasa -a 192.168.0.5 power off
kasa --address 192.168.0.5 metrics
*/

// Entry-point
func main() {

	// Values of the command-line flags, and the defaults
	flagAddress := ""
	flagPort := 9999
	flagInitialKey := 171
	flagFormat := "human"
	flagMetricsAddress := "127.0.0.1"
	flagMetricsPort := 5000
	flagMetricsPath := "/metrics"
	flagMetricsInterval := 15 // Default Prometheus scrape interval

	// Setup the command-line flags
	flag.StringVar( &flagAddress, "address", flagAddress, "The IPv4 address of the smart plug, e.g. 192.168.0.5." )
	flag.IntVar( &flagPort, "port", flagPort, "The port number for the smart plug API." )
	flag.IntVar( &flagInitialKey, "initial-key", flagInitialKey, "The initial value for the XOR encryption." )
	flag.StringVar( &flagFormat, "format", flagFormat, "The output format, either human-readable (human) or JSON (json)." )
	flag.StringVar( &flagMetricsAddress, "metrics-address", flagMetricsAddress, "The IPv4 address to listen on for the HTTP metrics server." )
	flag.IntVar( &flagMetricsPort, "metrics-port", flagMetricsPort, "The port number to listen on for the HTTP metrics server." )
	flag.StringVar( &flagMetricsPath, "metrics-path", flagMetricsPath, "The path to the metrics page." )
	flag.IntVar( &flagMetricsInterval, "metrics-interval", flagMetricsInterval, "The time in seconds to wait between collecting metrics." )

	// Set a custom help message
	flag.Usage = func() {
		fmt.Printf( "%s, v%s, by %s (%s).\n", PROJECT_NAME, PROJECT_VERSION, AUTHOR_NAME, AUTHOR_WEBSITE )
		fmt.Printf( "\nUsage: %s [-h/-help] [-address <IPv4 address>] [-port <number>] [-initial-key <number>] [-format <string>] [-metrics-address <IPv4 address>] [-metrics-port <number>] [-metrics-path <string>] [-metrics-interval <seconds>] [command] [argument, ...]\n", os.Args[ 0 ] )

		flag.PrintDefaults()

		fmt.Printf( "\nCommands: info, usage [now|total|average] [7d|30d], power [on|off], light [on|off], metrics\n" )

		os.Exit( 1 ) // By default it exits with code 2
	}

	// Parse the command-line flags
	flag.Parse()

	// Initial values for command-line arguments
	commandName := "info"
	commandArguments := []string{}

	// Parse the command-line arguments
	if ( len( flag.Args() ) > 0 ) {
		commandName = flag.Args()[ 0 ]

		if ( len( flag.Args() ) > 1 ) {
			commandArguments = flag.Args()[ 1 : ]
		}
	}

	// Ensure an IP address is provided
	if ( flagAddress == "" ) {
		exitWithErrorMessage( "The IPv4 address of the smart plug must be set using the -address flag, use -help for more information." )
	}

	// Require a valid IPv4 address for the smart plug
	plugAddress := net.ParseIP( flagAddress )
	if ( plugAddress == nil || plugAddress.To4() == nil ) {
		exitWithErrorMessage( "Invalid IPv4 address for smart plug." )
	}

	// Require a valid port number for the smart plug API
	if ( flagPort <= 0 || flagPort >= 65536 ) {
		exitWithErrorMessage( "Invalid port number for smart plug API, must be between 1 and 65535." )
	}

	// No need to check initial key as it can be any positive or negative integer

	// Require a valid output format
	if ( flagFormat != "human" && flagFormat != "json" ) {
		exitWithErrorMessage( "Invalid output format, must be either 'human' or 'json'." )
	}

	// Require a valid IPv4 address for the metrics server
	metricsAddress := net.ParseIP( flagMetricsAddress )
	if ( flagMetricsAddress == "" || metricsAddress == nil || metricsAddress.To4() == nil ) {
		exitWithErrorMessage( "Invalid listening IPv4 address for HTTP metrics server." )
	}

	// Require a valid port number for the metrics server
	if ( flagMetricsPort <= 0 || flagMetricsPort >= 65536 ) {
		exitWithErrorMessage( "Invalid listening port number for HTTP metrics server, must be between 1 and 65535." )
	}

	// Require a valid path for the metrics page
	if ( flagMetricsPath == "" || flagMetricsPath[ 0 : 1 ] != "/" || flagMetricsPath[ 1 : ] == "/" ) {
		exitWithErrorMessage( "Invalid path for the metrics page, must have a leading slash and no trailing slash." )
	}

	// Require a valid interval for collecting metrics
	if ( flagMetricsInterval <= 0 ) {
		exitWithErrorMessage( "Invalid interval to wait between collecting metrics, must be greater than 0." )
	}

	// Create the smart plug structure
	var smartPlug KasaSmartPlug

	// Connect to the smart plug
	connectError := smartPlug.Connect( plugAddress, flagPort, 5000 )
	if ( connectError != nil ) {
		exitWithErrorMessage( connectError.Error() )
	}

	// Disconnect from the smart plug once we're done
	defer smartPlug.Disconnect()

	// Set the initial encryption & decryption key
	smartPlug.InitialKey = flagInitialKey

	// Update all properties with latest data
	updatePropertiesError := smartPlug.UpdateProperties()
	if ( updatePropertiesError != nil ) {
		exitWithErrorMessage( updatePropertiesError.Error() )
	}

	// Is this execution for device information?
	if ( commandName == "info" ) {

		// Require no arguments
		if ( len( commandArguments ) > 0 ) {
			exitWithErrorMessage( "Information command does not require any arguments." )
		}

		// TODO: Display device information
		fmt.Printf( "Alias: '%s'.\n", smartPlug.Alias )
		fmt.Printf( "Icon: '%s'.\n", smartPlug.Icon )
		fmt.Printf( "Status: '%s'.\n", smartPlug.Status )
		fmt.Printf( "Uptime: '%d'.\n", smartPlug.Uptime )
		fmt.Printf( "Power State: '%t'.\n", smartPlug.PowerState )
		fmt.Printf( "Light State: '%t'.\n", smartPlug.LightState )
		fmt.Printf( "Device Name: '%s'.\n", smartPlug.DeviceName )
		fmt.Printf( "Device Model: '%s'.\n", smartPlug.DeviceModel )
		fmt.Printf( "Device Identifier: '%s'.\n", smartPlug.DeviceIdentifier )
		fmt.Printf( "Hardware Version: '%s'.\n", smartPlug.HardwareVersion )
		fmt.Printf( "Hardware Identifier: '%s'.\n", smartPlug.HardwareIdentifier )
		fmt.Printf( "Firmware Version: '%s'.\n", smartPlug.FirmwareVersion )
		fmt.Printf( "OEM Identifier: '%s'.\n", smartPlug.OEMIdentifier )
		fmt.Printf( "MAC Address: '%s'.\n", smartPlug.MACAddress )
		fmt.Printf( "Total Energy: '%d'.\n", smartPlug.Energy.Total )
		fmt.Printf( "Wattage: '%f'.\n", smartPlug.Energy.Wattage )
		fmt.Printf( "Voltage: '%f'.\n", smartPlug.Energy.Voltage )
		fmt.Printf( "Amperage: '%f'.\n", smartPlug.Energy.Amperage )
		fmt.Printf( "Signal Strength: '%d'.\n", smartPlug.SignalStrength )
		fmt.Printf( "Source: '%s'.\n", smartPlug.Source )
		fmt.Printf( "Type: '%s'.\n", smartPlug.Type )
		fmt.Printf( "NTC State: '%d'.\n", smartPlug.NTCState )

	// Is this execution for energy usage?
	} else if ( commandName == "usage" ) {

		// Defaults for optional arguments
		usageType := "now" // now, total, average
		usagePeriod := 30 // 7 (7 days), 30 (30 days)

		// Has an argument been provided?
		if ( len( commandArguments ) > 0 ) {

			// Set the energy usage type
			usageType = commandArguments[ 0 ]

			// Require a valid energy usage type
			if ( usageType != "now" && usageType != "total" && usageType != "average" ) {
				exitWithErrorMessage( "Unrecognised energy usage type, must be either 'now', 'total' or 'average'." )
			}

		}

		// Have extra arguments been provided?
		if ( len( commandArguments ) > 1 ) {

			// Fail if the energy usage type does not require a usage period
			if ( usageType == "now" ) {
				exitWithErrorMessage( "Energy usage type 'now' does not require an energy usage period." )
			}

			// Parse the energy usage period
			parsedPeriod, parseError := strconv.ParseInt( commandArguments[ 1 ], 10, 32 )
			if ( parseError != nil ) {
				fmt.Fprintf( os.Stderr, "Error while parsing energy usage period: '%s'\n", parseError )
				os.Exit( 1 )
			}

			// Require a valid energy usage period
			if ( parsedPeriod != 7 && parsedPeriod != 30 ) {
				exitWithErrorMessage( "Invalid energy usage period, must be either 7 or 30." )
			}

			// Set the energy usage period from a 64-bit to a regular integer
			usagePeriod = int( parsedPeriod )

		}

		// Have too many arguments been provided?
		if ( len( commandArguments ) > 2 ) {
			exitWithErrorMessage( "Energy usage command does not accept more than 2 arguments." )
		}

		// TODO: Display energy usage

		// Debugging
		fmt.Println( usageType, usagePeriod )

	// Is this execution to control the power relay?
	} else if ( commandName == "power" ) {

		// Require a single argument
		if ( len( commandArguments ) != 1 ) {
			exitWithErrorMessage( "Power command requires 1 argument for power state." )
		}

		// Parse the power state
		var powerState bool
		if ( commandArguments[ 0 ] == "on" ) {
			powerState = true
		} else if ( commandArguments[ 0 ] == "off" ) {
			powerState = false

		// Require a valid power state
		} else {
			exitWithErrorMessage( "Invalid power state, must be either 'on' or 'off'." )
		}

		// TODO: Set relay state

		// Debugging
		fmt.Println( powerState )

	// Is this execution to control the light?
	} else if ( commandName == "light" ) {

		// Require a single argument
		if ( len( commandArguments ) != 1 ) {
			exitWithErrorMessage( "Light command requires 1 argument for light state." )
		}

		// Parse the light state
		var lightState bool
		if ( commandArguments[ 0 ] == "on" ) {
			lightState = true
		} else if ( commandArguments[ 0 ] == "off" ) {
			lightState = false

		// Require a valid light state
		} else {
			exitWithErrorMessage( "Invalid light state, must be either 'on' or 'off'." )
		}

		// TODO: Set light state

		// Debugging
		fmt.Println( lightState )

	// Is this execution to serve metrics?
	} else if ( commandName == "metrics" ) {

		// Require no arguments
		if ( len( commandArguments ) > 0 ) {
			exitWithErrorMessage( "Metrics command does not require any arguments." )
		}

		// TODO: Start prometheus metrics server

	// Give help when a command does not exist
	} else {
		exitWithErrorMessage( "Unrecognised command, use -help for a list of commands." )
	}

}

// Displays a message on the standard error stream & exits with an failure status code
func exitWithErrorMessage( message string ) {
	fmt.Fprintln( os.Stderr, message )
	os.Exit( 1 )
}
