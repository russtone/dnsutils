package dnsutils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/russtone/dnsutils"
)

func TestSubdomains(t *testing.T) {
	dd := dnsutils.Subdomains("a.b.c.base.com", "base.com")
	assert.Equal(t, []string{
		"a.b.c.base.com",
		"b.c.base.com",
		"c.base.com",
	}, dd)
}
