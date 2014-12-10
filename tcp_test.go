package sunrpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewRecordMarker(t *testing.T) {
	assert.Equal(t, uint32(0), NewRecordMarker(0, false))
	assert.Equal(t, uint32(0x80000000), NewRecordMarker(0, true))
	assert.Equal(t, uint32(0x0002239E), NewRecordMarker(140190, false))
	assert.Equal(t, uint32(0x8002239E), NewRecordMarker(140190, true))
	assert.Equal(t, uint32(0x7FFFFFFF), NewRecordMarker(2147483647, false))
	assert.Equal(t, uint32(0xFFFFFFFF), NewRecordMarker(2147483647, true))
}

func TestParseRecordMarker(t *testing.T) {
	size, last := ParseRecordMarker(0)

	assert.Equal(t, 0, size)
	assert.False(t, last)

	size, last = ParseRecordMarker(0x80000000)

	assert.Equal(t, 0, size)
	assert.True(t, last)

	size, last = ParseRecordMarker(0x0002239E)

	assert.Equal(t, 140190, size)
	assert.False(t, last)

	size, last = ParseRecordMarker(0x8002239E)

	assert.Equal(t, 140190, size)
	assert.True(t, last)

	size, last = ParseRecordMarker(0x7FFFFFFF)

	assert.Equal(t, 2147483647, size)
	assert.False(t, last)

	size, last = ParseRecordMarker(0xFFFFFFFF)

	assert.Equal(t, 2147483647, size)
	assert.True(t, last)
}

func TestReadCall(t *testing.T) {
	buf := bytes.NewBuffer([]byte{
		0x38, 0x00, 0x00, 0x80, 0x54, 0x88, 0x7d, 0x26, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02,
		0x00, 0x01, 0x86, 0xa0, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x06, 0x00, 0x00, 0x00, 0x01,
	})

	mapping := mapping{}

	err := ReadCall(buf, &mapping)

	assert.Nil(t, err)
	assert.Equal(t, 1, mapping.Program)
	assert.Equal(t, 1, mapping.Version)
	assert.Equal(t, Tcp, mapping.Protocol)
	assert.Equal(t, 1, mapping.Port)
}
