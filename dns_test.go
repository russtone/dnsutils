package dnsutils_test

import (
	"fmt"
	"net"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/russtone/dnsutils"
)

func TestQuery(t *testing.T) {
	dnsIP := net.ParseIP("1.1.1.1")

	tests := []struct {
		name    string
		qtype   string
		results []string
	}{
		{"dns.google", "A", []string{"8.8.4.4", "8.8.8.8"}},
		{"dns.yandex", "A", []string{"77.88.8.8"}},
		{"dns.google", "AAAA", []string{"2001:4860:4860::8844", "2001:4860:4860::8888"}},
		{"yandex.ru", "MX", []string{"mx.yandex.ru"}},
		{"google.com", "NS", []string{
			"ns1.google.com",
			"ns2.google.com",
			"ns3.google.com",
			"ns4.google.com",
		}},
		{"_spf.yandex.ru", "TXT", []string{"v=spf1 include:_spf-ipv4.yandex.ru include:_spf-ipv6.yandex.ru include:_spf-ipv4-yc.yandex.ru ~all"}},
		{"_caldavs._tcp.yandex.ru", "SRV", []string{"caldav.yandex.ru"}},
		{"www.twitter.com", "CNAME", []string{"twitter.com"}},
		{dnsutils.PTR(net.ParseIP("1.1.1.1")), "PTR", []string{"one.one.one.one"}},
		{dnsutils.PTR(net.ParseIP("2606:4700:4700::1111")), "PTR", []string{"one.one.one.one"}},
	}

	for _, vec := range tests {
		t.Run(fmt.Sprintf("%s-%s", vec.name, vec.qtype), func(t *testing.T) {
			rr, err := dnsutils.Query(dnsIP, vec.name, vec.qtype)
			assert.NoError(t, err)
			sort.StringSlice(rr).Sort()
			assert.Equal(t, rr, vec.results)
		})
	}
}
