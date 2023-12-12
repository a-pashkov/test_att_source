package app

import (
	"encoding/hex"
	"fmt"
	"internal/navigation"
	"internal/ndtp"
	"internal/wialon"
	"net"
	"os"
	"strconv"
	"time"
)

func Start() {

	if len(os.Args) != 7 {
		usageAndExit()
	}

	ip := os.Args[1]
	if net.ParseIP(ip) == nil {
		fmt.Fprintf(os.Stderr, "IP must be valid IPv4 address\n")
		usageAndExit()
	}

	port, err := strconv.Atoi(os.Args[2])
	if err != nil || port < 0 || port > 65535 {
		fmt.Fprintf(os.Stderr, "PORT must be 0 - 65535\n")
		usageAndExit()
	}

	id, err := strconv.Atoi(os.Args[4])
	if err != nil {
		fmt.Fprintf(os.Stderr, "ID must be integer\n")
		usageAndExit()
	}

	lat, err := strconv.ParseFloat(os.Args[5], 32)
	if err != nil || lat < -90.0 || lat > 90.0 {
		fmt.Fprintf(os.Stderr, "LAT must be -90.0 - 90.0\n")
		usageAndExit()
	}

	lon, err := strconv.ParseFloat(os.Args[6], 32)
	if err != nil || lon < -180 || lon > 180 {
		fmt.Fprintf(os.Stderr, "LON must be -180 - 180\n")
		usageAndExit()
	}

	tm := time.Now()

	p := navigation.Packet{AttId: uint32(id), Time: tm, Lat: lat, Lon: lon}

	ptype := os.Args[3]
	var client navigation.NavProto
	switch ptype {
	case "ndtp":
		client = &ndtp.Ndtp{}
	case "wialon":
		client = &wialon.Wialon{}
	default:
		fmt.Fprintf(os.Stderr, "Wrong TYPE: %s\n", ptype)
		usageAndExit()

	}

	fmt.Println("Data:", p)

	conn, err := net.Dial("tcp", net.JoinHostPort("127.0.0.1", "4444"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Connection error: %s\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	bData, err := client.Send(&p, conn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Send error: %s\n", err)
		os.Exit(1)
	}
	showBinPacks(bData)
}

func usageAndExit() {
	msg := "Usage: rnis_protocols_emulator IP PORT TYPE ID LAT LON\n" +
		"TYPE can be 'ndtp','wialon'"
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}

func showBinPacks(data *[][]byte) {
	for i, d := range *data {
		fmt.Printf("Packet %d:\n%s", i, hex.Dump(d))
	}
}
