package ndtp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"internal/navigation"
	"math"
	"net"

	"github.com/sigurn/crc16"
)

var nplSignature = []byte{0x7E, 0x7E}

const (
	nplAddressServer      = 0x00000000
	nplHeaderLen          = 15
	nphHeaderLen          = 10
	nphSrvGenericControls = 0
	nphSrvNavdata         = 1
	nphSgcConnRequest     = 100
	nphSndRealtime        = 101
)

type Ndtp struct {
	packetId uint16
}

func (n *Ndtp) formNpl(nph *[]byte) *[]byte {
	size := len(*nph)
	table16 := crc16.MakeTable(crc16.CRC16_MODBUS)
	cs := crc16.Checksum(*nph, table16)

	npl := make([]byte, nplHeaderLen)
	copy(npl[0:2], nplSignature)
	binary.LittleEndian.PutUint16(npl[2:4], uint16(size))
	binary.LittleEndian.PutUint16(npl[4:6], 0x02) // Flags
	binary.BigEndian.PutUint16(npl[6:8], cs)
	npl[8] = 0x01 // Type
	binary.LittleEndian.PutUint32(npl[9:13], nplAddressServer)
	binary.LittleEndian.PutUint16(npl[13:15], n.packetId)

	n.packetId += 1

	return &npl
}

func (n *Ndtp) formNphAuth(p *navigation.Packet) *[]byte {
	nphHeader := make([]byte, nphHeaderLen)
	binary.LittleEndian.PutUint16(nphHeader[0:2], nphSrvGenericControls) // NPH_SRV_GENERIC_CONTROLS = 0
	binary.LittleEndian.PutUint16(nphHeader[2:4], nphSgcConnRequest)     // NPH_SGC_CONN_REQUEST = 100
	binary.LittleEndian.PutUint16(nphHeader[4:6], 0x00)                  // Flags
	binary.LittleEndian.PutUint16(nphHeader[6:10], n.packetId)           // RequestID

	nphData := make([]byte, 14)
	binary.LittleEndian.PutUint16(nphData[0:2], 6)               // _VerHi
	binary.LittleEndian.PutUint16(nphData[2:4], 2)               // _VerLo
	binary.BigEndian.PutUint16(nphData[4:6], 0b0000000100000010) // _Flags
	binary.LittleEndian.PutUint32(nphData[6:10], p.AttId)        // PeerID
	binary.LittleEndian.PutUint32(nphData[10:14], 0x00000400)    // MaxSize

	nph := append(nphHeader, nphData...)
	return &nph
}

func (n *Ndtp) formNphNavdata(p *navigation.Packet) *[]byte {
	nphHeader := make([]byte, nphHeaderLen)
	binary.LittleEndian.PutUint16(nphHeader[0:2], nphSrvNavdata)
	binary.LittleEndian.PutUint16(nphHeader[2:4], nphSndRealtime)
	binary.LittleEndian.PutUint16(nphHeader[4:6], 0x00)        // Flags
	binary.LittleEndian.PutUint16(nphHeader[6:10], n.packetId) // RequestID

	nphData := make([]byte, 28)
	nphData[0] = 0x00 // Cell type
	nphData[1] = 0x00 // Source GPS
	binary.LittleEndian.PutUint32(nphData[2:6], uint32(p.Time.Unix()))
	binary.LittleEndian.PutUint32(nphData[6:10], uint32(math.Abs(p.Lon)*10000000))
	binary.LittleEndian.PutUint32(nphData[10:14], uint32(math.Abs(p.Lat)*10000000))

	var flags byte = 0x80 // val, lonHs, latHs, accum, first, sos, alarms, call
	if p.Lon >= 0 {
		flags |= 0x01 << 0x06
	}
	if p.Lon >= 0 {
		flags |= 0x01 << 0x05
	}
	nphData[14] = flags

	nph := append(nphHeader, nphData...)
	return &nph
}

func (n *Ndtp) Send(p *navigation.Packet, conn net.Conn) (*[][]byte, error) {
	var bdata [][]byte
	var msg []byte
	var npl, nph *[]byte
	if n.packetId == 0 {
		nph = n.formNphAuth(p)
		npl = n.formNpl(nph)
		msg = append(*npl, *nph...)

		_, err := conn.Write(msg)
		if err != nil {
			return nil, err
		}

		resp := make([]byte, 256)
		c, err := conn.Read(resp)
		if err != nil {
			return nil, err
		}
		err = checkResponse(resp)
		if err != nil {
			fmt.Println("Resp1:", resp[:c])
		}

		bdata = append(bdata, msg)
	}

	nph = n.formNphNavdata(p)
	npl = n.formNpl(nph)
	msg = append(*npl, *nph...)

	_, err := conn.Write(msg)
	if err != nil {
		return nil, err
	}

	resp := make([]byte, 256)
	c, err := conn.Read(resp)
	if err != nil {
		return nil, err
	}
	err = checkResponse(resp)
	if err != nil {
		fmt.Println("Resp2:", resp[:c])
	}

	bdata = append(bdata, msg)
	return &bdata, nil
}

func checkResponse(r []byte) error {
	temp := []byte{
		// NPL
		126, 126, // signature
		14, 0, // data_size
		2, 0, // flags
		1, 171, // crc
		2,          // type
		0, 0, 0, 0, // peer_address
		0, 0, // request_id
		// NPH
		0, 0, // service_id
		0, 0, // type
		0, 0, // flags
		0, 0, 0, 0, // request_id

		0, 0, 0, 0} // error code
	if bytes.Equal(r, temp) {
		return errors.New("wrong response")
	}

	return nil
}
