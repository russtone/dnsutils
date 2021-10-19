package dnsutils

import (
	"net"
	"time"
)

// server represents DNS server.
type server struct {
	// IP address of the server.
	ip net.IP

	// Count of processed queries.
	queriesCount int

	// Creation time.
	createdAt time.Time

	// Last usage time.
	lastUsedAt time.Time
}

// query makes DNS-query and returns its result.
func (s *server) query(name string, qtype string) ([]string, error) {

	defer func() {
		s.queriesCount++
		s.lastUsedAt = time.Now()
	}()

	res, err := Query(name, s.ip, qtype)

	if err != nil {
		return nil, err
	}

	return res, err
}

// rate returns average queries per second for the server.
func (s *server) rate() float64 {
	return float64(s.queriesCount) / float64(time.Since(s.createdAt).Seconds())
}

// delay returns the time to wait to slow down the request rate to required limit.
func (s *server) delay(rateLimit float64) time.Duration {
	return time.Duration(1/rateLimit)*time.Second - time.Since(s.lastUsedAt)
}

// pool represents pool of DNS servers.
type pool struct {

	// Ready DNS servers.
	servers chan *server

	// Rate limit (queries per second) for servers in pool.
	rateLimit float64
}

// newPool creates new DNS servers pool.
func newPool(ips []net.IP, rateLimit float64) *pool {
	servers := make(chan *server, len(ips))

	for _, ip := range ips {
		servers <- &server{ip: ip, createdAt: time.Now()}
	}

	return &pool{
		servers:   servers,
		rateLimit: rateLimit,
	}
}

// take returns ready DNS server from pool.
func (p *pool) take() *server {
	return <-p.servers
}

// release returns the server to the pool.
func (p *pool) release(s *server) {
	go func() {
		if delay := s.delay(p.rateLimit); delay > 0 {
			time.Sleep(delay)
		}

		p.servers <- s
	}()
}
