package main

import (
	"flag"
	"fmt"
	"os"
)

// REQUIRED is the value of the flag that is required.
const REQUIRED = "_REQUIRED"

var grpcURL = "localhost:6666"

func mainHost(hostFlag *flag.FlagSet) {
	hostFlag.Parse(os.Args[2:])
	addr := hostFlag.Lookup("addr").Value.String()
	username := hostFlag.Lookup("username").Value.String()
	password := hostFlag.Lookup("password").Value.String()

	fmt.Println(addr, username, password)

	requiredFlag := []string{"username", "password"}
	for _, flagName := range requiredFlag {
		if hostFlag.Lookup(flagName).Value.String() == REQUIRED {
			fmt.Printf("error while parsing arguments: -%s is required\n", flagName)
			hostFlag.Usage()
			os.Exit(1)
		}
	}
	startHost(addr, username, password)
}

func mainClient(clientFlag *flag.FlagSet) {
	clientFlag.Parse(os.Args[2:])
	addr := clientFlag.Lookup("addr").Value.String()
	username := clientFlag.Lookup("username").Value.String()
	password := clientFlag.Lookup("password").Value.String()
	token := clientFlag.Lookup("token").Value.String()
	requiredFlag := []string{"username", "password", "token"}
	for _, flagName := range requiredFlag {
		if clientFlag.Lookup(flagName).Value.String() == REQUIRED {
			fmt.Printf("error while parsing arguments: -%s is required\n", flagName)
			clientFlag.Usage()
			os.Exit(1)
		}
	}
	startClient(addr, username, password, token)
}

func initClientFlag() *flag.FlagSet {
	addr, username, password, hostToken := new(string), new(string), new(string), new(string)
	clientCmd := flag.NewFlagSet("client", flag.ExitOnError)
	clientCmd.StringVar(&grpcURL, "server", grpcURL, "the address of the signaling server")
	clientCmd.StringVar(addr, "addr", "localhost:8000", "the address of the server")
	clientCmd.StringVar(username, "username", REQUIRED, "the username of the client")
	clientCmd.StringVar(password, "password", REQUIRED, "the password of the client")
	clientCmd.StringVar(hostToken, "token", `REQUIRED`, "the token of the host")
	return clientCmd

}

func initHostFlag() *flag.FlagSet {
	hostCmd := flag.NewFlagSet("host", flag.ExitOnError)
	hostCmd.StringVar(&grpcURL, "server", grpcURL, "the address of the signaling server")
	hostCmd.StringVar(new(string), "addr", "localhost:8000", "the address of the server")
	hostCmd.StringVar(new(string), "username", REQUIRED, "the username of the host")
	hostCmd.StringVar(new(string), "password", REQUIRED, "the password of the host")

	return hostCmd

}

func init() {
	flag.Usage = func() {
		_, _ = fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s <host|client> [flags]\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	// parse flags
	// usage: yamcli <host|client> [flags]
	// where host is the host mode and client is the client mode.
	// parse host or client and than pass to the corresponding function.

	// space bwteen log params
	if len(os.Args) < 2 {
		flag.Usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "host":
		hostFlag := initHostFlag()
		mainHost(hostFlag)
	case "client":
		clientFlag := initClientFlag()
		mainClient(clientFlag)
	default:
		flag.Usage()
		os.Exit(1)
	}
}
