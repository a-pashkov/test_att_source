package navigation

import (
	"fmt"
	"net"
	"time"
)

type Packet struct {
	AttId uint32
	Time  time.Time
	Lon   float64
	Lat   float64
}

func (p Packet) String() string {
	s := fmt.Sprintf(`{"id": %d, "time": "%s", "lat": %f, "lon": %f} `, p.AttId, p.Time.Format("2006-01-02 15:04:05"), p.Lat, p.Lon)
	return s
}

type NavProto interface {
	Send(*Packet, net.Conn) (*[][]byte, error)
}
