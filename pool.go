package dnsutils

import (
	"fmt"
	"math/rand"
	"net"
	"time"
)

// Server represents DNS Server.
type Server struct {
	// IP address of the Server.
	IP net.IP

	// queriesCount of processed queries.
	queriesCount int

	// createdAt is time of creation.
	createdAt time.Time

	// lastUsedAt is time of last usage.
	lastUsedAt time.Time
}

// Query makes DNS query and returns its result.
func (s *Server) Query(name string, qtype string) ([]string, error) {
	defer func() {
		s.queriesCount++
		s.lastUsedAt = time.Now()
	}()

	res, err := Query(s.IP, name, qtype)

	if err != nil {
		return nil, err
	}

	return res, err
}

// Rate returns average queries per second for the Server.
func (s *Server) Rate() float64 {
	return float64(s.queriesCount) / time.Since(s.createdAt).Seconds()
}

// Pool represents Pool of DNS servers.
type Pool struct {
	// servers are list of available DNS servers..
	servers []*Server

	// ready is channel of ready to use servers.
	ready chan *Server

	// rate is rate limit (query per second)
	rate int

	// ticker internal time.Ticker using for refill.
	ticker *time.Ticker

	// stop signal channel to stop refill.
	stop chan struct{}

	// rnd is random.
	rnd *rand.Rand
}

// NewPool returns new instane of Pool.
func NewPool(ips []net.IP, rate, capacity int) *Pool {
	if rate <= 0 {
		panic(fmt.Sprintf("rate is %d", rate))
	}

	if len(ips) == 0 {
		panic("empty ips")
	}

	servers := make([]*Server, 0)
	mincap := rate * len(ips)

	if capacity < mincap {
		capacity = mincap
	}

	ready := make(chan *Server, capacity)

	for _, ip := range ips {
		servers = append(servers, &Server{IP: ip, createdAt: time.Now()})
	}

	for i := 0; i < mincap; i++ {
		ready <- servers[i%len(servers)]
	}

	return &Pool{
		servers: servers,
		ready:   ready,
		rate:    rate,
		rnd:     rand.New(rand.NewSource(time.Now().Unix())),
	}
}

// Start start internal refill goroutine.
func (p *Pool) Start() {
	p.ticker = time.NewTicker(time.Second)
	p.stop = make(chan struct{})

	go func() {
		defer close(p.stop)

		for {
			select {
			case <-p.ticker.C:
				p.fill()
			case <-p.stop:
				p.ticker.Stop()
				return
			}
		}
	}()
}

// Close closes all internal resources.
func (p *Pool) Close() {
	p.stop <- struct{}{}
	<-p.stop
	close(p.ready)
}

// Take returns ready DNS Server from Pool.
func (p *Pool) Take() *Server {
	return <-p.ready
}

// fill fills ready servers.
func (p *Pool) fill() {
	n := len(p.servers)
	for i := 0; i < p.rate; i++ {
		off := p.rnd.Intn(len(p.servers))
		for j := 0; j < n; j++ {
			select {
			case p.ready <- p.servers[(off+j)%n]:
			default:
			}
		}
	}
}
