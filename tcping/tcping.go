package tcping

import (
	"fmt"
	"net"
	"time"
)

func Test(host string, port int) (bool, float64, error) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 5*time.Second)

	if err != nil {
		return false, 0, err
	}

	defer conn.Close()
	duration := time.Since(start).Seconds()

	return true, duration, nil
}
