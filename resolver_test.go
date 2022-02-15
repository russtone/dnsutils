package dnsutils_test

import (
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/russtone/dnsutils"
)

func TestResolver(t *testing.T) {

	tests := []struct {
		name    string
		qtypes  []string
		results map[string][]string
	}{
		{"dns.google", []string{"A"}, map[string][]string{"A": {"8.8.4.4", "8.8.8.8"}}},
		{"dns.yandex", []string{"A"}, map[string][]string{"A": {"77.88.8.8"}}},
	}

	r := dnsutils.NewResolver(
		[]net.IP{
			net.ParseIP("1.1.1.1"),
			net.ParseIP("8.8.8.8"),
		}, 2, 1)

	r.Start()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		var (
			res  dnsutils.Result
			err  error
			done int
		)

		for r.Next(&res, &err) {
			done++

			for _, tt := range tests {
				if tt.name == res.Name {
					res.SortAnswers()
					assert.EqualValues(t, tt.results, res.Answers)
				}
			}
		}

		assert.Equal(t, len(tests), done)
	}()

	for _, tt := range tests {
		r.Add(tt.name, tt.qtypes, nil)
	}

	r.Wait()
	r.Close()

	wg.Wait()
}
