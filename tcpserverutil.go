package sunrpc

import (
	"bytes"
	"encoding/binary"
	"io"
	"io/ioutil"

	"github.com/dropbox/godropbox/errors"
)

const (
	maxRecordSize = 32 * 1024
)

// NewRecordMarker creates a new record marker as described in RFC 5531.
//
// "When RPC messages are passed on top of a byte stream transport protocol (like TCP), it is
// necessary to delimit one message from another in order to detect and possibly recover from
// protocol errors. This is called record marking (RM). One RPC message fits into one RM record."
//
// The first argument is the size of the subsequent RPC message, the second argument denotes whether
// this marker denotes the last record in this transmission.
//
// See also RFC 5531, Section 11: https://tools.ietf.org/html/rfc5531#section-11
func NewRecordMarker(size uint32, last bool) uint32 {
	marker := size
	marker &^= 1 << 31

	if last {
		marker ^= 0x80000000
	}

	return marker
}

// ParseRecordMarker deconstructs a record marker returning the record size and whether the given
// marker denotes the last frame of an RPC message.
func ParseRecordMarker(marker uint32) (size uint32, last bool) {
	size = marker &^ (1 << 31)
	last = (marker >> 31) == 1

	return size, last
}

// ReadRecordMarker reads the record marker from the given Reader with the appropriate endianness.
func ReadRecordMarker(r io.Reader) (size uint32, last bool, err error) {
	var marker uint32

	if err := binary.Read(r, binary.BigEndian, &marker); err != nil {
		return 0, true, err
	}

	size, last = ParseRecordMarker(marker)

	return size, last, nil
}

// WriteRecordMarker writes the a record marker to a Writer with the given size and "last fragment"
// indicator with the appropriate endianness.
func WriteRecordMarker(w io.Writer, size uint32, last bool) error {
	record := NewRecordMarker(uint32(size), true)

	if err := binary.Write(w, binary.BigEndian, record); err != nil {
		return err
	}

	return nil
}

// ReadRecord reads a whole record into memory (up to 32 KB), otherwise the record is discarded.
func ReadRecord(r io.Reader) (*bytes.Buffer, error) {
	size, last, err := ReadRecordMarker(r)

	if err != nil {
		return nil, errors.Wrap(err, "Could not read the TCP record marker")
	}

	if size < 1 {
		return nil, errors.New("A TCP record must be at least one byte in size")
	}

	if !last {
		return nil, errors.New("Records composed of multiple fragments are not supported yet")
	}

	if size >= maxRecordSize {
		io.CopyN(ioutil.Discard, r, int64(size))
	}

	var buf bytes.Buffer

	buf.Grow(int(size))
	io.CopyN(&buf, r, int64(size))

	return &buf, nil
}

// WriteTCPReplyMessage writes an outgoing "reply" message with the appropriate framing structure
// required by RPC-over-TCP.
func WriteTCPReplyMessage(w io.Writer, xid uint32, ret interface{}) error {
	// Buffer reply data so that we can compute a proper record marker later on
	var buf bytes.Buffer

	size, err := WriteReplyMessage(&buf, xid, ret)
	if err != nil {
		return errors.Wrap(err, "Could not write the reply message to a buffer")
	}

	// Write the record marker
	//
	// FIXME: Assuming we are sending a single record
	if err := WriteRecordMarker(w, uint32(size), true); err != nil {
		return errors.Wrap(err, "Could not write the record marker to the given io.Writer")
	}

	// Write the payload
	if _, err := w.Write(buf.Bytes()); err != nil {
		return errors.Wrap(err, "Could not write the record payload to the given io.Writer")
	}

	return nil
}
