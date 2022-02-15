package dnsutils

import (
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

	// rnd random.
	rnd *rand.Rand
}

// NewPool returns new instane of Pool.
func NewPool(ips []net.IP, rate int, capacity int) *Pool {
	servers := make([]*Server, 0)
	ready := make(chan *Server, capacity)
	mincap := rate * len(ips)

	// guard
	if capacity < mincap {
		capacity = mincap
	}

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
}

// Stop stops internal refill goroutine.
func (p *Pool) Stop() {
	p.stop <- struct{}{}
	<-p.stop
}

// Take returns ready DNS Server from Pool.
func (p *Pool) Take() *Server {
	return <-p.ready
}

// fill fills ready servers.
func (p *Pool) fill() {
	for i := 0; i < p.rate; i++ {
		for _, j := range p.rnd.Perm(len(p.servers)) {
			select {
			case p.ready <- p.servers[j]:
			default:
			}
		}
	}
}
