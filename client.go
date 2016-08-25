package sunrpc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/rasky/go-xdr/xdr2"
)

const ClientMaxRpcMessageSize = 32 * 1024

type ClientTransport uint32

const (
	ClientTransportTcpUdp  ClientTransport = iota // first try TCP, fallback to UDP
	ClientTransportUdpTcp                         // first try UDP, fallback to TCP
	ClientTransportTcpOnly                        // TCP only
	ClientTransportUdpOnly                        // UDP only
)

type ClientConfig struct {
	Transport ClientTransport // transport to use (default: ClientTransportTcpUdp)
	Timeout   time.Duration   // read/write timeout (default: 5 seconds)
}

type Client struct {
	Addr    string
	Program uint32
	Version uint32
	cfg     ClientConfig

	mu           sync.Mutex
	conn         net.Conn
	disconnected bool
}

var clientBufPool = sync.Pool{
	New: func() interface{} {
		data := make([]byte, ClientMaxRpcMessageSize)
		return &data
	},
}

// NewClient creates a new RPC client. The client will connect to a RPC server at the specified
// address (in net.Dial format), and will talk to the specified program/version service.
// cfg contains the optional configuration for this client.
// This function does not attempt any connection; the client will lazily connect (and possibly error out)
// when Call() is first called. You can call proc #0 (always reserved as ping) if you need to check
// the presence of the service.
func NewClient(addr string, program, version uint32, cfg *ClientConfig) *Client {
	if cfg == nil {
		cfg = &ClientConfig{}
	}
	var zz time.Duration
	if cfg.Timeout == zz {
		cfg.Timeout = 5 * time.Second
	}

	return &Client{
		Addr:         addr,
		Program:      program,
		Version:      version,
		cfg:          *cfg,
		disconnected: true,
	}
}

func (c *Client) Close() {
	c.mu.Lock()
	c.close()
	c.mu.Unlock()
}

// Call the specified proc in the RPC server, optionally passing some args, and receive
// the reply body in reply.
//
// On top of network errors, err can be one of the errors defined in this package to signal
// specific error conditions that callers might want to specifically handle.
func (c *Client) Call(proc uint32, args, reply interface{}) (err error) {
	return c.CallProgram(c.Program, c.Version, proc, args, reply)
}

// CallProgram is like Call, but allows to define a non-default program and version.
func (c *Client) CallProgram(program, version uint32, proc uint32, args, reply interface{}) error {
	if c.disconnected {
		if err := c.reconnect(); err != nil {
			return err
		}
		if proc == 0 {
			// we already executed a ping during reconnection, so don't send a second one
			return nil
		}
	}

	var useUdp bool
	var buf bytes.Buffer

	_, useUdp = c.conn.(*net.UDPConn)

	pcall := NewProcedureCall(program, version, proc)
	if _, err := xdr.Marshal(&buf, pcall); err != nil {
		return err
	}

	// Write procedure arguments to the buffer (if any)
	if args != nil {
		if _, err := xdr.Marshal(&buf, args); err != nil {
			return err
		}
	}

	// Set write timeout to avoid stalling forever
	var zd time.Duration
	if c.cfg.Timeout != zd {
		c.conn.SetWriteDeadline(time.Now().Add(c.cfg.Timeout))
	}

	// On TCP transport, we need to write a record marker
	if !useUdp {
		// Because of a bug on the Linux implementation of rpcbind, we want
		// to send the record marker and the payload in a single TCP segment
		// if possible (so with a single conn.Write)
		full := bytes.NewBuffer(make([]byte, 0, buf.Len()+4))
		if err := WriteRecordMarker(full, uint32(buf.Len()), true); err != nil {
			return err
		}
		io.Copy(full, &buf)

		// Send the payload
		if _, err := c.conn.Write(full.Bytes()); err != nil {
			c.disconnected = true
			return err
		}
	} else {
		// Send the payload
		if _, err := c.conn.Write(buf.Bytes()); err != nil {
			c.disconnected = true
			return err
		}
	}

	// Read the reply header. We want this to happen in a pure network
	// read so that we can detect whether the server is actually replying
	// or there is a network error (specifically important in case of UDP:
	// in fact, in that case, this is where we get an error if the UDP port
	// was closed while sending).
	var replyh ProcedureReply

	if c.cfg.Timeout != zd {
		c.conn.SetReadDeadline(time.Now().Add(c.cfg.Timeout))
	}

	var reader io.Reader

	if _, ok := c.conn.(*net.UDPConn); !ok {
		// On TCP transport, we need to read the record through different markers
		if buf, err := ReadRecord(c.conn); err != nil {
			c.disconnected = true
			return err
		} else {
			reader = buf
		}
	} else {
		// On UDP, we need to read the whole answer through a single Read()
		// call because it is a single datagram. Use a pool of buffers
		// to speed up processing
		buf := clientBufPool.Get().(*[]byte)
		defer clientBufPool.Put(buf)

		if n, err := c.conn.Read(*buf); err != nil {
			c.disconnected = true
			return err
		} else {
			reader = bytes.NewReader((*buf)[:n])
		}
	}

	if _, err := xdr.Unmarshal(reader, &replyh); err != nil {
		return err
	}

	if replyh.Header.Xid != pcall.Header.Xid {
		return errors.New("invalid Xid in reply")
	}

	if replyh.Header.Type != Reply {
		return errors.New("invalid reply type")
	}

	if replyh.Type != Accepted {
		switch replyh.Rejected.Stat {
		case RpcMismatch:
			return &ErrRpcMismatch{High: replyh.Rejected.MismatchInfo.High, Low: replyh.Rejected.MismatchInfo.Low}
		case AuthError:
			return &ErrAuth{Stat: replyh.Rejected.AuthStat}
		default:
			c.disconnected = true
			return fmt.Errorf("RPC reply has invalid wire format")
		}
	}

	if replyh.Accepted.Stat != Success {
		switch replyh.Accepted.Stat {
		case ProgMismatch:
			return &ErrProgMismatch{High: replyh.Accepted.MismatchInfo.High, Low: replyh.Accepted.MismatchInfo.Low}
		case ProcUnavail:
			return &ErrProcUnavail{}
		case ProgUnavail:
			return &ErrProgUnavail{}
		case GarbageArgs:
			return &ErrGarbageArgs{}
		}
	}

	// Everything is OK, read reply body (if any)
	if reply != nil {
		if _, err := xdr.Unmarshal(reader, reply); err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) close() {
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	c.disconnected = true
}

func (c *Client) reconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.close()

	var prot []string
	switch c.cfg.Transport {
	case ClientTransportTcpUdp:
		prot = []string{"tcp", "udp"}
	case ClientTransportUdpTcp:
		prot = []string{"udp", "tcp"}
	case ClientTransportUdpOnly:
		prot = []string{"udp"}
	case ClientTransportTcpOnly:
		prot = []string{"tcp"}
	}

	for _, p := range prot {
		conn, err := net.Dial(p, c.Addr)
		if err == nil {
			c.conn = conn
			c.disconnected = false
			// Check with procedure 0, which is always reserved as a ping
			if c.Call(0, nil, nil) == nil {
				return nil
			}
			c.conn = nil
			c.disconnected = true
			conn.Close()
		}
	}

	return errors.New("cannot connect to RPC server")
}
