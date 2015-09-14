package sunrpc

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/rasky/go-xdr/xdr2"
)

// WriteCall writes an RPC "call" message to the given writer in order to call a remote procedure
// with the given program, version and procedure identifiers. Args holds the arguments to pass to
// the remote procedure.
func WriteCall(w io.Writer, program uint32, version uint32, proc uint32, args interface{}) error {
	var buf bytes.Buffer

	if _, err := xdr.Marshal(&buf, NewProcedureCall(program, version, proc)); err != nil {
		return err
	}

	// Write procedure arguments to the buffer
	if _, err := xdr.Marshal(&buf, args); err != nil {
		return err
	}

	// Write the record marker before sending the payload
	bytes := buf.Bytes()

	if err := binary.Write(w, binary.LittleEndian, NewRecordMarker(uint32(len(bytes)), true)); err != nil {
		return err
	}

	// Send the payload
	if _, err := w.Write(bytes); err != nil {
		return err
	}

	return nil
}
