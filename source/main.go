package main

import (
	"flag"
	"fmt"
	"net"
	"os"
)

const (
	PROJECT_NAME = "Kasa Smart Plug"
	PROJECT_VERSION = "2.0.0"

	AUTHOR_NAME = "viral32111"
	AUTHOR_WEBSITE = "https://viral32111.com"
)

/*
kasa [-h/--help] [-a/--address <IPv4 address>] [-p/--port <number (def. 9999)>] [-k/--initial-key <number (def. 171)>] [-f/--format <human|json (def. human)>] [--metrics-address <IPv4 address (def. 127.0.0.1)>] [--metrics-port <number (def. 5000)>] [--metrics-path <string (def. /metrics)>] [-i/--metrics-interval <seconds (def. 15)>] [command] [argument, ...]

commands: info, usage [now|total|average] [7d|30d], power [on|off], light [on|off], metrics

kasa -a 192.168.0.5
kasa -a 192.168.0.5 info
kasa -a 192.168.0.5 -f json uptime
kasa --address 192.168.0.5 --port 9999 usage
kasa -a 192.168.0.5 -p 9999 power on
kasa -a 192.168.0.5 power off
kasa --address 192.168.0.5 metrics
*/

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
	flag.StringVar( &flagMetricsAddress, "metrics-address", flagMetricsAddress, "The IP address to listen on for the HTTP metrics server." )
	flag.IntVar( &flagMetricsPort, "metrics-port", flagMetricsPort, "The port number to listen on for the HTTP metrics server." )
	flag.StringVar( &flagMetricsPath, "metrics-path", flagMetricsPath, "The path to the metrics page." )
	flag.IntVar( &flagMetricsInterval, "metrics-interval", flagMetricsInterval, "The time in seconds to wait between collecting metrics." )

	// Set a custom help message
	flag.Usage = func() {
		fmt.Printf( "%s, v%s, by %s (%s).\n", PROJECT_NAME, PROJECT_VERSION, AUTHOR_NAME, AUTHOR_WEBSITE )

		fmt.Printf( "\nUsage: kasa [-h/-help] [-address <IPv4 address>] [-port <number>] [-initial-key <number>] [-format <string>] [-metrics-address <IPv4 address>] [-metrics-port <number>] [-metrics-path <string>] [-metrics-interval <seconds>] [command] [argument, ...]\n" )
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
		fmt.Fprintln( os.Stderr, "The IPv4 address of the smart plug must be set using the -address flag, see -help for more information." )
		os.Exit( 1 )
	}

	// Require a valid IP address for the smart plug
	plugAddress := net.ParseIP( flagAddress )
	if ( plugAddress == nil || plugAddress.To4() == nil ) {
		fmt.Fprintln( os.Stderr, "Invalid IPv4 address for smart plug." )
		os.Exit( 1 )
	}

	// Require a valid IP address for the smart plug API
	if ( flagPort <= 0 || flagPort >= 65536 ) {
		fmt.Fprintln( os.Stderr, "Invalid port number for smart plug API, must be between 1 and 65535." )
		os.Exit( 1 )
	}

	// No need to check initial key as it can be any positive or negative integer

	// Require a valid output format
	if ( flagFormat != "human" && flagFormat != "json" ) {
		fmt.Fprintln( os.Stderr, "Invalid output format, must be either 'human' or 'json'." )
		os.Exit( 1 )
	}

	// Require a valid IP address for the metrics server
	metricsAddress := net.ParseIP( flagMetricsAddress )
	if ( flagMetricsAddress == "" || metricsAddress == nil || metricsAddress.To4() == nil ) {
		fmt.Fprintln( os.Stderr, "Invalid listening IPv4 address for HTTP metrics server." )
		os.Exit( 1 )
	}

	// Require a valid IP address for the metrics server
	if ( flagMetricsPort <= 0 || flagMetricsPort >= 65536 ) {
		fmt.Fprintln( os.Stderr, "Invalid listening port number for HTTP metrics server, must be between 1 and 65535." )
		os.Exit( 1 )
	}

	// Require a valid path for the metrics page
	if ( flagMetricsPath == "" || flagMetricsPath[ 0 : 1 ] != "/" ) {
		fmt.Fprintln( os.Stderr, "Invalid path for the metrics page." )
		os.Exit( 1 )
	}

	// Require a valid interval for collecting metrics
	if ( flagMetricsInterval <= 0 ) {
		fmt.Fprintln( os.Stderr, "Invalid interval to wait between collecting metrics, must be greater than 0." )
		os.Exit( 1 )
	}

	// Debugging
	fmt.Println( commandName, commandArguments )

}
