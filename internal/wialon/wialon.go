package wialon

import (
	"bytes"
	"errors"
	"fmt"
	"internal/navigation"
	"math"
	"net"
	"strings"
)

type Wialon struct {
	auth bool
}

func (n *Wialon) Send(p *navigation.Packet, conn net.Conn) ([][]byte, error) {
	var bdata [][]byte
	if !n.auth {
		msg := []byte(fmt.Sprintf("#L#%d;NA\r\n", p.AttId))
		_, err := conn.Write(msg)
		if err != nil {
			return nil, err
		}
		bdata = append(bdata, msg)

		resp := make([]byte, 256)
		c, err := conn.Read(resp)
		if err != nil {
			return nil, err
		}

		err = checkResponse(resp[:c])
		if err != nil {
			return nil, err
		}
	}

	dt := p.Time.UTC().Format("020106")
	tm := p.Time.UTC().Format("150405")

	lat1 := fmt.Sprintf("%09.4f", math.Abs(p.Lat*100))
	lat2 := "N"
	if p.Lat < 0 {
		lat2 = "S"
	}

	lon1 := fmt.Sprintf("%010.4f", math.Abs(p.Lon*100))
	lon2 := "E"
	if p.Lon < 0 {
		lon2 = "W"
	}

	var speed, course, height, sats string = "NA", "NA", "NA", "NA"

	msg := []byte("#SD#" + strings.Join([]string{dt, tm, lat1, lat2, lon1, lon2, speed, course, height, sats}, ";") + "\r\n")
	_, err := conn.Write(msg)
	if err != nil {
		return nil, err
	}
	bdata = append(bdata, msg)

	resp := make([]byte, 256)
	c, err := conn.Read(resp)
	if err != nil {
		return nil, err
	}

	err = checkResponse(resp[:c])
	if err != nil {
		return nil, err
	}

	return bdata, nil
}

func checkResponse(r []byte) error {
	authOk := []byte("#AL#1\r\n")
	dataOk := []byte("#ASD#1\r\n")
	if !bytes.Equal(r, authOk) && !bytes.Equal(r, dataOk) {
		return errors.New("wrong response")
	}

	return nil
}
