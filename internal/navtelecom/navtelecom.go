package navtelecom

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"internal/navigation"
	"math"
	"net"
)

const (
	authBodyLen = 19
)

type Navtelecom struct {
	auth   bool
	recNum uint32
}

func (n *Navtelecom) Send(p *navigation.Packet, conn net.Conn) ([][]byte, error) {
	var bdata [][]byte
	var body, header, packet []byte

	if !n.auth {
		body = n.formNavtelecomAuth(p)
		header = n.formNavtelecomHeader(body)
		packet = append(header, body...)

		_, err := conn.Write(packet)
		if err != nil {
			return nil, err
		}
		bdata = append(bdata, packet)

		resp := make([]byte, 256)
		c, err := conn.Read(resp)
		if err != nil {
			return nil, err
		}

		err = checkResponse(resp[:c])
		if err != nil {
			return nil, err
		}
		n.auth = true
	}

	body = n.formNavtelecomData(p)
	header = n.formNavtelecomHeader(body)
	packet = append(header, body...)

	_, err := conn.Write(packet)
	if err != nil {
		return nil, err
	}
	bdata = append(bdata, packet)

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

func (n *Navtelecom) formNavtelecomHeader(body []byte) []byte {
	header := []byte{
		0x40, 0x4e, 0x54, 0x43, // "@NTC"
		0x01, 0x00, 0x00, 0x00, // IDr
		0x00, 0x00, 0x00, 0x00, // IDs
		0x00, 0x00, // Body size
		0x00, // Data CS
		0x18} // Header CS

	bodyLen := len(body)
	binary.LittleEndian.PutUint16(header[12:14], uint16(bodyLen))

	header[14] = bxorChecksum(body)
	header[15] = bxorChecksum(header[0:15])

	return header
}

func (n *Navtelecom) formNavtelecomAuth(p *navigation.Packet) []byte {
	auth := make([]byte, authBodyLen)
	copy(auth[0:4], "*>S:")

	imei := []byte(fmt.Sprintf("%015d", p.AttId))
	copy(auth[4:19], imei)
	return auth
}

func (n *Navtelecom) formNavtelecomData(p *navigation.Packet) []byte {
	data := make([]byte, 75)
	copy(data[0:3], []byte("*>T"))

	binary.LittleEndian.PutUint16(data[3:5], 0x02) // type

	binary.BigEndian.PutUint32(data[5:9], n.recNum)
	n.recNum++

	utcTime := p.Time.UTC()

	data[11] = byte(utcTime.Hour())
	data[12] = byte(utcTime.Minute())
	data[13] = byte(utcTime.Second())
	data[14] = byte(utcTime.Day())
	data[15] = byte(utcTime.Month())
	data[16] = byte(utcTime.Year() - 2000)

	data[43] = byte(utcTime.Hour())
	data[44] = byte(utcTime.Minute())
	data[45] = byte(utcTime.Second())
	data[46] = byte(utcTime.Day())
	data[47] = byte(utcTime.Month())
	data[48] = byte(utcTime.Year() - 2000)

	binary.BigEndian.PutUint32(data[49:53], math.Float32bits(float32(p.Lat)))
	binary.BigEndian.PutUint32(data[53:57], math.Float32bits(float32(p.Lon)))

	return data
}

func bxorChecksum(data []byte) (checksum byte) {
	for _, d := range data {
		checksum ^= d
	}
	return
}

func checkResponse(r []byte) error {

	authOk := []byte{64, 78, 84, 67, 0, 0, 0, 0, 1, 0, 0, 0, 3, 0, 69, 94, 42, 60, 83}
	dataOk := []byte{64, 78, 84, 67, 0, 0, 0, 0, 1, 0, 0, 0, 7, 0, 66, 93, 42, 60, 84, 0, 0, 0, 0}
	if !bytes.Equal(r, authOk) && !bytes.Equal(r, dataOk) {
		return errors.New("wrong response")
	}

	return nil
}
