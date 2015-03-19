package sunrpc

import (
	"bytes"
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

	assert.EqualValues(t, 0, size)
	assert.False(t, last)

	size, last = ParseRecordMarker(0x80000000)

	assert.EqualValues(t, 0, size)
	assert.True(t, last)

	size, last = ParseRecordMarker(0x0002239E)

	assert.EqualValues(t, 140190, size)
	assert.False(t, last)

	size, last = ParseRecordMarker(0x8002239E)

	assert.EqualValues(t, 140190, size)
	assert.True(t, last)

	size, last = ParseRecordMarker(0x7FFFFFFF)

	assert.EqualValues(t, 2147483647, size)
	assert.False(t, last)

	size, last = ParseRecordMarker(0xFFFFFFFF)

	assert.EqualValues(t, 2147483647, size)
	assert.True(t, last)
}

func TestReadRecordMarker(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0x80, 0x00, 0x00, 0x38})

	size, last, err := ReadRecordMarker(buf)

	assert.EqualValues(t, 56, size)
	assert.True(t, last)
	assert.Nil(t, err)
}

func TestWriteRecordMarker(t *testing.T) {
	var buf bytes.Buffer

	expected := []byte{0x80, 0x00, 0x00, 0x38}

	err := WriteRecordMarker(&buf, 56, true)

	assert.Nil(t, err)
	assert.Equal(t, expected, buf.Bytes())
}

func TestReadTCPCallMessage(t *testing.T) {
	buf := bytes.NewBuffer([]byte{
		0x80, 0x00, 0x00, 0x28 /**/, 0x54, 0x88, 0x7d, 0x26,
		0x00, 0x00, 0x00, 0x00 /**/, 0x00, 0x00, 0x00, 0x02,
		0x00, 0x01, 0x86, 0xa0 /**/, 0x00, 0x00, 0x00, 0x02,
		0x00, 0x00, 0x00, 0x01 /**/, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00 /**/, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
	})

	recordBuf, err := ReadRecord(buf)
	call, err := ReadProcedureCall(recordBuf)

	assert.Nil(t, err)
	assert.EqualValues(t, PortmapperProgram, call.Body.Program)
	assert.EqualValues(t, PortmapperVersion, call.Body.Version)
	assert.EqualValues(t, PortmapperPortSet, call.Body.Procedure)
}
