package tcptlsutil

import (
	"crypto/tls"
	"errors"
	"net"
	"time"
)

type TcpTlsConn struct {
	// tcp: true
	// tls: false
	isTcpConn         bool
	tcpConn           *net.TCPConn
	tlsConn           *tls.Conn
	isClose           bool
	nextConnectPolicy int
}

func NewFromTcpConn(tcpConn *net.TCPConn) (c *TcpTlsConn) {
	c = &TcpTlsConn{}
	c.tcpConn = tcpConn
	c.isTcpConn = true
	c.isClose = false
	c.nextConnectPolicy = NEXT_CONNECT_POLICY_KEEP
	return c
}

func NewFromTlsConn(tlsConn *tls.Conn) (c *TcpTlsConn) {
	c = &TcpTlsConn{}
	c.tlsConn = tlsConn
	c.isTcpConn = false
	c.isClose = false
	c.nextConnectPolicy = NEXT_CONNECT_POLICY_KEEP
	return c
}

func (c *TcpTlsConn) RemoteAddr() net.Addr {
	if c.isTcpConn && c.tcpConn != nil {
		return c.tcpConn.RemoteAddr()
	} else if c.tlsConn != nil {
		return c.tlsConn.RemoteAddr()
	}
	return nil
}

func (c *TcpTlsConn) LocalAddr() net.Addr {
	if c.isTcpConn && c.tcpConn != nil {
		return c.tcpConn.LocalAddr()
	} else if c.tlsConn != nil {
		return c.tlsConn.LocalAddr()
	}
	return nil
}

func (c *TcpTlsConn) Write(b []byte) (n int, err error) {
	if c.isTcpConn && c.tcpConn != nil && !c.isClose {
		return c.tcpConn.Write(b)
	} else if c.tlsConn != nil && !c.isClose {
		return c.tlsConn.Write(b)
	}
	return -1, errors.New("is not conn")
}

func (c *TcpTlsConn) Read(b []byte) (n int, err error) {
	if c.isTcpConn && c.tcpConn != nil && !c.isClose {
		return c.tcpConn.Read(b)
	} else if c.tlsConn != nil && !c.isClose {
		return c.tlsConn.Read(b)
	}
	return -1, errors.New("is not conn")
}

func (c *TcpTlsConn) Close() (err error) {
	if c.isTcpConn && c.tcpConn != nil && !c.isClose {
		c.isClose = true
		return c.tcpConn.Close()
	} else if c.tlsConn != nil && !c.isClose {
		c.isClose = true
		return c.tlsConn.Close()
	}
	return errors.New("is not conn")
}

func (c *TcpTlsConn) SetDeadline(t time.Time) error {
	if c.isTcpConn && c.tcpConn != nil {
		return c.tcpConn.SetDeadline(t)
	} else if c.tlsConn != nil {
		return c.tlsConn.SetDeadline(t)
	}
	return errors.New("is not conn")
}

func (c *TcpTlsConn) IsClose() bool {
	return c.isClose
}