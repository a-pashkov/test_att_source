package egtsdisp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"internal/navigation"
	"math"
	"net"

	"github.com/sigurn/crc16"
	"github.com/sigurn/crc8"
)

type EgtsDisp struct {
	pid uint16
	rid uint16
}

func (n *EgtsDisp) Send(p *navigation.Packet, conn net.Conn) ([][]byte, error) {
	var bdata [][]byte

	msg := n.formEgtsPacket(p)

	_, err := conn.Write(msg)
	if err != nil {
		return nil, err
	}

	resp := make([]byte, 256)
	c, err := conn.Read(resp)
	if err != nil {
		return nil, err
	}
	err = checkResponse(resp[:c])
	if err != nil {
		return nil, err
	}

	bdata = append(bdata, msg)
	return bdata, nil
}

func (e *EgtsDisp) formEgtsPacket(p *navigation.Packet) []byte {
	rec := e.formEgtsRecord(p)

	packet := make([]byte, 11+len(rec)+2)

	packet[0] = 0x01 //PRV
	packet[1] = 0x00 //SKID
	packet[2] = 0x03 //FLAGS
	packet[3] = 0x0b //HL
	packet[4] = 0x00 //HE

	fdl := len(rec)
	binary.LittleEndian.PutUint16(packet[5:7], uint16(fdl))

	binary.LittleEndian.PutUint16(packet[7:9], uint16(e.pid))
	e.pid += 1

	packet[9] = 1 //PT

	params8 := crc8.Params{Poly: 0x31, Init: 0xff, RefIn: false, RefOut: false, XorOut: 0x00, Check: 0xf7, Name: "rnis_crc8"}
	table8 := crc8.MakeTable(params8)
	hcs := crc8.Checksum(packet[0:10], table8)
	packet[10] = hcs

	copy(packet[11:], rec)

	params16 := crc16.CRC16_CCITT_FALSE
	table16 := crc16.MakeTable(params16)
	sfrcs := crc16.Checksum(rec, table16)
	binary.LittleEndian.PutUint16(packet[11+len(rec):], sfrcs)

	return packet
}

func (e *EgtsDisp) formEgtsRecord(p *navigation.Packet) []byte {
	srData := e.formEgtsSrPosData(p)
	sr := e.formEgtsSr(0x10, srData)

	record := make([]byte, 11+len(sr))

	rl := uint16(len(sr))
	binary.LittleEndian.PutUint16(record[0:2], rl)

	rn := uint16(e.rid)
	binary.LittleEndian.PutUint16(record[2:4], rn)
	e.rid += 1

	rfl := byte(0x01) // OBFE = 1
	record[4] = rfl

	binary.LittleEndian.PutUint32(record[5:9], uint32(p.AttId))

	sst := byte(0x02)
	record[9] = sst

	rst := byte(0x02)
	record[10] = rst

	copy(record[11:], sr)

	return record
}

func (e *EgtsDisp) formEgtsSr(t uint8, d []byte) []byte {
	srl := len(d)
	sr := make([]byte, srl+3)
	sr[0] = byte(t)
	binary.LittleEndian.PutUint16(sr[1:3], uint16(srl))
	copy(sr[3:], d)

	return sr
}

func (e *EgtsDisp) formEgtsSrPosData(p *navigation.Packet) []byte {
	srBody := make([]byte, 21)

	ntm := uint32(p.Time.Unix() - 1262304000)
	binary.LittleEndian.PutUint32(srBody[0:4], ntm)

	lat := uint32(math.Abs(p.Lat) / 90 * 0xffffffff)
	binary.LittleEndian.PutUint32(srBody[4:8], lat)

	lon := uint32(math.Abs(p.Lon) / 180 * 0xffffffff)
	binary.LittleEndian.PutUint32(srBody[8:12], lon)

	var flags, alte, lohs, lahs, mv, bb, cs, fix, vld byte
	if p.Lon < 0 {
		lohs = 1
	}
	if p.Lat < 0 {
		lahs = 1
	}
	fix = 1
	vld = 1
	flags = alte << 0x07
	flags |= lohs << 0x06
	flags |= lahs << 0x05
	flags |= mv << 0x04
	flags |= bb << 0x03
	flags |= cs << 0x02
	flags |= fix << 0x01
	flags |= vld

	srBody[12] = flags

	spd := 0.0
	spd_hi := byte(math.Abs(spd) * 10 / 256)
	spd_lo := byte(uint16(math.Abs(spd)*10) % 256)
	dir := 0.0
	bear_hi := byte(math.Abs(dir) / 256)
	bear_lo := byte(uint16(math.Abs(dir)) % 256)

	var alts, flags2 byte
	flags2 = (bear_hi << 0x07) | ((alts << 0x06) & 0x40) | (spd_hi & 0x3F)

	srBody[13] = spd_lo
	srBody[14] = flags2
	srBody[15] = bear_lo

	var odometer = []byte{0, 0, 0}
	copy(srBody[16:19], odometer)

	var din, src byte
	srBody[19] = din
	srBody[20] = src

	return srBody
}

func checkResponse(r []byte) error {
	respOk := []byte{
		0x01,       // PRV
		0x00,       // SKID
		0x03,       // Flags
		0x0b,       // HL
		0x00,       // HE
		0x10, 0x00, // FDL
		0x00, 0x00, // PID
		0x00, // PT
		0xb3, // HCS

		0x00, 0x00, // RPID (Response Packet ID)
		0x00, // PR (Processing Result)

		0x06, 0x00, // RL
		0x00, 0x00, // RN
		0x18, // Flags 0b00011000
		0x02, // SST
		0x02, // RST

		0x00,       // SRT (Subrecord Туре)
		0x03, 0x00, // SRL (Subrecord Length)
		0x00, 0x00, // CRN (Confirmed Record Number)
		0x00, // RST (Record Status)

		0xf7, 0x9e} // SFRCS}

	if !bytes.Equal(r, respOk) {
		return errors.New("wrong response")
	}

	return nil
}
