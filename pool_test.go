package dnsutils_test

import (
	"net"
	"testing"
	"time"

	"github.com/russtone/dnsutils"
	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {

	servers := []net.IP{
		net.ParseIP("1.1.1.1"),
		net.ParseIP("2.2.2.2"),
		net.ParseIP("3.3.3.3"),
	}

	pool := dnsutils.NewPool(servers, 5, 15)
	go pool.Start()

	ss := make([]*dnsutils.Server, 0)
	stop := make(chan struct{})
	ticks := time.Tick(10 * time.Millisecond)
	timeout := time.After(2 * time.Second)

	go func() {
		defer close(stop)

		for {
			select {
			case <-ticks:
				ss = append(ss, pool.Take())
			case <-timeout:
				return
			}
		}
	}()

	<-stop

	pool.Stop()

	assert.GreaterOrEqual(t, len(ss), 30)
	assert.LessOrEqual(t, len(ss), 40)
}
