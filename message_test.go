package sunrpc

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReadArguments(t *testing.T) {
	buf := bytes.NewBuffer([]byte{
		0x00, 0x00, 0x00, 0x01 /**/, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x06 /**/, 0x00, 0x00, 0x00, 0x01,
	})

	mapping := mapping{}

	err := ReadArguments(buf, &mapping)

	assert.Nil(t, err)
	assert.Equal(t, 1, mapping.Program)
	assert.Equal(t, 1, mapping.Version)
	assert.Equal(t, Tcp, mapping.Protocol)
	assert.Equal(t, 1, mapping.Port)
}

func TestReadArgumentsMissingData(t *testing.T) {
	buf := bytes.NewBuffer([]byte{
		0x00, 0x00, 0x00, 0x01 /**/, 0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x06 /**/, 0x00, 0x00,
	})

	mapping := mapping{}
	err := ReadArguments(buf, &mapping)

	assert.NotNil(t, err)
}
