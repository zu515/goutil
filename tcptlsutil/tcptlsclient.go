package tcptlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/cpusoft/goutil/belogs"
	"github.com/cpusoft/goutil/convert"
	"github.com/cpusoft/goutil/jsonutil"
	"github.com/cpusoft/goutil/osutil"
)

type TcpTlsClient struct {
	// both tcp and tls
	isTcpClient         bool
	tcpTlsClientProcess TcpTlsClientProcess

	// for tls
	tlsRootCrtFileName    string
	tlsPublicCrtFileName  string
	tlsPrivateKeyFileName string

	// for close
	tcpTlsConn *TcpTlsConn

	// for channel
	TcpTlsMsg chan TcpTlsMsg
}

// server: 0.0.0.0:port
func NewTcpClient(tcpTlsClientProcess TcpTlsClientProcess, tcpTlsMsg chan TcpTlsMsg) (tc *TcpTlsClient) {

	belogs.Debug("NewTcpClient():tcpTlsClientProcess:", tcpTlsClientProcess)
	tc = &TcpTlsClient{}
	tc.isTcpClient = true
	tc.tcpTlsClientProcess = tcpTlsClientProcess
	tc.TcpTlsMsg = tcpTlsMsg
	belogs.Info("NewTcpClient():tc:", tc)
	return tc
}

// server: 0.0.0.0:port
func NewTlsClient(tlsRootCrtFileName, tlsPublicCrtFileName, tlsPrivateKeyFileName string,
	tcpTlsClientProcess TcpTlsClientProcess, tcpTlsMsg chan TcpTlsMsg) (tc *TcpTlsClient, err error) {

	belogs.Debug("NewTlsClient():tcpTlsClientProcess:", &tcpTlsClientProcess)
	tc = &TcpTlsClient{}
	tc.isTcpClient = false
	tc.tcpTlsClientProcess = tcpTlsClientProcess
	tc.TcpTlsMsg = tcpTlsMsg

	rootExists, _ := osutil.IsExists(tlsRootCrtFileName)
	if !rootExists {
		belogs.Error("NewTlsClient():root cer files not exists:", tlsRootCrtFileName)
		return nil, errors.New("root cer file is not exists")
	}
	publicExists, _ := osutil.IsExists(tlsPublicCrtFileName)
	if !publicExists {
		belogs.Error("NewTlsClient():public cer files not exists:", tlsPublicCrtFileName)
		return nil, errors.New("public cer file is not exists")
	}
	privateExists, _ := osutil.IsExists(tlsPrivateKeyFileName)
	if !privateExists {
		belogs.Error("NewTlsClient():private cer files not exists:", tlsPrivateKeyFileName)
		return nil, errors.New("private cer file is not exists")
	}

	tc.tlsRootCrtFileName = tlsRootCrtFileName
	tc.tlsPublicCrtFileName = tlsPublicCrtFileName
	tc.tlsPrivateKeyFileName = tlsPrivateKeyFileName

	belogs.Info("NewTlsClient():tc:", &tc)
	return tc, nil
}

// server: **.**.**.**:port
func (tc *TcpTlsClient) StartTcpClient(server string) (err error) {
	belogs.Debug("StartTcpClient(): create client, server is  ", server)

	conn, err := net.DialTimeout("tcp", server, 60*time.Second)
	if err != nil {
		belogs.Error("StartTcpClient(): DialTimeout fail, server:", server, err)
		return err
	}
	belogs.Debug("StartTcpClient(): DialTimeout ok, server is  ", server)

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		belogs.Error("StartTcpClient(): conn cannot conver to tcpConn: ", conn.RemoteAddr().String(), err)
		return err
	}
	belogs.Debug("StartTcpClient(): tcpConn ok, server is  ", server)

	tc.tcpTlsConn = NewFromTcpConn(tcpConn)
	//active send to server, and receive from server, loop
	belogs.Debug("StartTcpClient(): NewFromTcpConn ok, server:", server, "   tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String())
	go tc.waitTcpTlsMsg()

	tc.onConnect()
	belogs.Info("StartTcpClient(): onConnect, server is  ", server, "  tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String())

	return nil
}

// server: **.**.**.**:port
func (tc *TcpTlsClient) StartTlsClient(server string) (err error) {
	belogs.Debug("StartTlsClient(): create client, server is  ", server,
		"  tlsPublicCrtFileName:", tc.tlsPublicCrtFileName,
		"  tlsPrivateKeyFileName:", tc.tlsPrivateKeyFileName)

	cert, err := tls.LoadX509KeyPair(tc.tlsPublicCrtFileName, tc.tlsPrivateKeyFileName)
	if err != nil {
		belogs.Error("StartTlsClient(): LoadX509KeyPair fail: server:", server,
			"  tlsPublicCrtFileName:", tc.tlsPublicCrtFileName,
			"  tlsPrivateKeyFileName:", tc.tlsPrivateKeyFileName, err)
		return err
	}
	belogs.Debug("StartTlsClient(): LoadX509KeyPair ok, server is  ", server)

	rootCrtBytes, err := ioutil.ReadFile(tc.tlsRootCrtFileName)
	if err != nil {
		belogs.Error("StartTlsClient(): ReadFile tlsRootCrtFileName fail, server:", server,
			"  tlsRootCrtFileName:", tc.tlsRootCrtFileName, err)
		return err
	}
	belogs.Debug("StartTlsClient(): ReadFile ok, server is  ", server)

	rootCertPool := x509.NewCertPool()
	ok := rootCertPool.AppendCertsFromPEM(rootCrtBytes)
	if !ok {
		belogs.Error("StartTlsClient(): AppendCertsFromPEM tlsRootCrtFileName fail,server:", server,
			"  tlsRootCrtFileName:", tc.tlsRootCrtFileName, "  len(rootCrtBytes):", len(rootCrtBytes), err)
		return err
	}
	belogs.Debug("StartTlsClient(): AppendCertsFromPEM ok, server is  ", server)

	config := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            rootCertPool,
		InsecureSkipVerify: false,
	}
	dialer := &net.Dialer{Timeout: time.Duration(60) * time.Second}
	tlsConn, err := tls.DialWithDialer(dialer, "tcp", server, config)
	if err != nil {
		belogs.Error("StartTlsClient(): DialWithDialer fail, server:", server, err)
		return err
	}
	belogs.Debug("StartTlsClient(): DialWithDialer ok, server is  ", server)

	/*
		tlsConn, err := tls.Dial("tcp", server, config)
		if err != nil {
			belogs.Error("StartTlsClient(): Dial fail, server:", server, err)
			return err
		}
	*/
	tc.tcpTlsConn = NewFromTlsConn(tlsConn)
	belogs.Debug("StartTlsClient(): NewFromTlsConn ok, server:", server, "   tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String())
	//active send to server, and receive from server, loop
	go tc.waitTcpTlsMsg()

	tc.onConnect()
	belogs.Info("StartTlsClient(): onConnect, server is  ", server, "  tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String())

	return nil
}

func (tc *TcpTlsClient) onConnect() {
	// call process func onConnect
	tc.tcpTlsClientProcess.OnConnectProcess(tc.tcpTlsConn)
	belogs.Info("onConnect(): tcptlsclient  after OnConnectProcess, tcpTlsConn: ", tc.tcpTlsConn.RemoteAddr().String())
}

func (tc *TcpTlsClient) onClose() {
	// close in the end
	belogs.Info("onClose(): tcptlsclient , tcpTlsConn: ", tc.tcpTlsConn.RemoteAddr().String())
	tc.tcpTlsClientProcess.OnCloseProcess(tc.tcpTlsConn)
	tc.tcpTlsConn.Close()

}

func (tc *TcpTlsClient) waitTcpTlsMsg() (err error) {
	belogs.Debug("waitTcpTlsMsg(): tcptlsclient , tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String())
	for {
		// wait next tcpTlsMsg: only error or NEXT_CONNECT_POLICY_CLOSE_** will end loop
		select {
		case tcpTlsMsg := <-tc.TcpTlsMsg:
			belogs.Info("waitTcpTlsMsg(): tcpTlsMsg:", jsonutil.MarshalJson(tcpTlsMsg),
				"  tcpTlsConn: ", tc.tcpTlsConn.RemoteAddr().String())

			switch tcpTlsMsg.MsgType {
			case MSG_TYPE_CLIENT_CLOSE_CONNECT:
				belogs.Info("waitTcpTlsMsg(): tcptlsclient msgType is MSG_TYPE_CLIENT_CLOSE_CONNECT,",
					" will close for tcpTlsConn: ", tc.tcpTlsConn.RemoteAddr().String())
				tc.onClose()
				// end for/select
				return nil
			case MSG_TYPE_ACTIVE_SEND_DATA:
				belogs.Info("waitTcpTlsMsg(): tcptlsclient msgType is MSG_TYPE_ACTIVE_SEND_DATA,",
					" will send to tcpTlsConn: ", tc.tcpTlsConn.RemoteAddr().String())
				nextConnectClosePolicy := tcpTlsMsg.NextConnectClosePolicy
				nextRwPolicy := tcpTlsMsg.NextRwPolicy
				sendData := tcpTlsMsg.SendData
				belogs.Debug("waitTcpTlsMsg(): tcptlsclient  send to server:", tc.tcpTlsConn.RemoteAddr().String(),
					"   nextConnectClosePolicy: ", nextConnectClosePolicy,
					"   nextRwPolicy:", nextRwPolicy,
					"   sendData:", convert.PrintBytesOneLine(sendData))

				// send data
				start := time.Now()
				n, err := tc.tcpTlsConn.Write(sendData)
				if err != nil {
					belogs.Error("waitTcpTlsMsg(): tcptlsclient   Write fail:  tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String(), err)
					return err
				}
				belogs.Debug("waitTcpTlsMsg(): tcptlsclient   Write to tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String(),
					"  len(sendData):", len(sendData), "  write n:", n, "   nextRwPolicy:", nextRwPolicy,
					"  time(s):", time.Since(start))

				// if wait receive, then wait next tcpTlsMsg
				if nextRwPolicy == NEXT_RW_POLICY_WAIT_READ {
					// if server tell client: end this loop, or end conn
					err := tc.onReceive()
					if err != nil {
						belogs.Error("waitTcpTlsMsg(): tcptlsclient   Write fail:  tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String(), err)
						return err
					}
					belogs.Info("waitTcpTlsMsg(): tcptlsclient NEXT_RW_POLICY_WAIT_READ, onReceive, tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String(),
						"  len(sendData):", len(sendData), "  write n:", n,
						"  time(s):", time.Since(start))
					continue
				} else {
					belogs.Info("waitTcpTlsMsg(): tcptlsclient no NEXT_RW_POLICY_WAIT_READ, onReceive, will return: ", tc.tcpTlsConn.RemoteAddr().String())
					continue
				}
			}
		}
	}

}

func (tc *TcpTlsClient) onReceive() (err error) {
	belogs.Debug("onReceive(): tcptlsclient  wait for onReceive, tcpTlsConn:", tc.tcpTlsConn.RemoteAddr().String())
	var leftData []byte
	// one packet
	buffer := make([]byte, 2048)
	// wait for new packet to read

	for {
		start := time.Now()
		n, err := tc.tcpTlsConn.Read(buffer)
		//	if n == 0 {
		//		continue
		//	}
		if err != nil {
			if err == io.EOF {
				// is not error, just client close
				belogs.Debug("onReceive(): tcptlsclient   io.EOF, client close: ", tc.tcpTlsConn.RemoteAddr().String(), err)
				return nil
			}
			belogs.Error("onReceive(): tcptlsclient   Read fail, err ", tc.tcpTlsConn.RemoteAddr().String(), err)
			return err
		}

		belogs.Debug("onReceive(): tcptlsclient, tcpTlsConn: ", tc.tcpTlsConn.RemoteAddr().String(),
			"  Read n", n, "  time(s):", time.Now().Sub(start))
		nextRwPolicy, leftData, err := tc.tcpTlsClientProcess.OnReceiveProcess(tc.tcpTlsConn, append(leftData, buffer[:n]...))
		belogs.Info("onReceive(): tcptlsclient  tcpTlsClientProcess.OnReceiveProcess, tcpTlsConn: ", tc.tcpTlsConn.RemoteAddr().String(), " receive n: ", n,
			"  len(leftData):", len(leftData), "  nextRwPolicy:", nextRwPolicy, "  time(s):", time.Now().Sub(start))
		if err != nil {
			belogs.Error("onReceive(): tcptlsclient  tcpTlsClientProcess.OnReceiveProcess  fail ,will close this tcpTlsConn : ", tc.tcpTlsConn.RemoteAddr().String(), err)
			return err
		}
		if nextRwPolicy == NEXT_RW_POLICY_END_READ {
			belogs.Debug("onReceive(): tcptlsclient  nextRwPolicy, will end this write/read loop: ", tc.tcpTlsConn.RemoteAddr().String())
			return nil
		}

	}

}

func (tc *TcpTlsClient) SendMsg(tcpTlsMsg *TcpTlsMsg) {

	belogs.Debug("SendMsg(): tcptlsclient, tcpTlsMsg:", jsonutil.MarshalJson(*tcpTlsMsg))
	tc.TcpTlsMsg <- *tcpTlsMsg
}

func (tc *TcpTlsClient) SendMsgForCloseConnect() {
	// send channel, and wait listener and conns end itself process and close loop
	belogs.Info("SendMsgForCloseConnect(): tcptlsclient will close graceful")
	tcpTlsMsg := &TcpTlsMsg{
		MsgType: MSG_TYPE_CLIENT_CLOSE_CONNECT,
	}
	tc.SendMsg(tcpTlsMsg)
}