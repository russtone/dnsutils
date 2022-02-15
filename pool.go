package dnsutils

import (
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

// Delay returns the time to wait to slow down the request Rate to required limit.
func (s *Server) Delay(rateLimit float64) time.Duration {
	return time.Duration(1/rateLimit)*time.Second - time.Since(s.lastUsedAt)
}

// Pool represents Pool of DNS servers.
type Pool struct {

	// Ready DNS servers.
	servers chan *Server

	// Rate limit (queries per second) for servers in Pool.
	rateLimit float64
}

// NewPool creates new DNS servers Pool.
func NewPool(ips []net.IP, rateLimit float64) *Pool {
	servers := make(chan *Server, len(ips))

	for _, ip := range ips {
		servers <- &Server{IP: ip, createdAt: time.Now()}
	}

	return &Pool{
		servers:   servers,
		rateLimit: rateLimit,
	}
}

// Take returns ready DNS Server from Pool.
func (p *Pool) Take() *Server {
	return <-p.servers
}

// Release returns the Server to the Pool.
func (p *Pool) Release(s *Server) {
	go func() {
		if delay := s.Delay(p.rateLimit); delay > 0 {
			time.Sleep(delay)
		}

		p.servers <- s
	}()
}
