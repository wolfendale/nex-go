package nex

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/PretendoNetwork/nex-go/v2/constants"
)

// TODO separate client from transport like kinnay/NintendoClients so that local ports make sense

type TestPRUDPClient struct {
	server          *PRUDPServer
	conn            *net.UDPConn
	sequenceCounter *SequenceCounter
	State           ConnectionState
	StreamType      constants.StreamType
	LocalPort       uint8
	RemotePort      uint8
}

// Starts a UDP connection to the server and performs an initial handshake, setting the State to Connected
func (tpc *TestPRUDPClient) Start() error {

	if tpc.State != StateNotConnected {
		return fmt.Errorf("Invalid state to start connection: %v", tpc.State)
	}

	tpc.State = StateConnecting

	var err error
	addr := server.udpSocket.LocalAddr().(*net.UDPAddr)
	tpc.conn, err = net.DialUDP("udp", nil, addr)
	if err != nil {
		return err
	}

	err = tpc.syn()
	if err != nil {
		return err
	}

	err = tpc.connect()
	if err != nil {
		return err
	}

	tpc.State = StateConnected

	// TODO ping keepalive

	return nil
}

// Sends a SYN packet and waits for an ACK
func (tpc *TestPRUDPClient) syn() error {

	syn := PRUDPPacketV0{PRUDPPacket{server: tpc.server}}
	syn.SetSourceVirtualPortStreamID(tpc.LocalPort)
	syn.SetSourceVirtualPortStreamType(tpc.StreamType)
	syn.SetDestinationVirtualPortStreamID(tpc.RemotePort)
	syn.SetDestinationVirtualPortStreamType(tpc.StreamType)
	syn.SetType(constants.SynPacket)
	syn.AddFlag(constants.PacketFlagNeedsAck)
	syn.SetSequenceID(1)
	syn.setSignature(syn.calculateSignature([]byte{}, []byte{}))
	connectionSignature, _ := syn.calculateConnectionSignature(server.udpSocket.LocalAddr())
	syn.setConnectionSignature(connectionSignature)

	_, err := tpc.conn.Write(syn.Bytes())
	if err != nil {
		return err
	}

	buf := make([]byte, server.FragmentSize)
	tpc.conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err = tpc.conn.Read(buf)
	if err != nil {
		return err
	}

	readStream := NewByteStreamIn(buf, server.LibraryVersions, tpc.server.ByteStreamSettings)
	ack, err := NewPRUDPPacketV0(tpc.server, nil, readStream)
	if err != nil {
		return err
	}

	if !(ack.HasFlag(constants.PacketFlagAck) && ack.Type() == constants.SynPacket && ack.sequenceID == syn.sequenceID) {
		return fmt.Errorf("Expected syn ack, got: %v", ack)
	}

	return nil
}

// Sends a CONNECT packet and waits for an ACK
func (tpc *TestPRUDPClient) connect() error {

	connect := PRUDPPacketV0{PRUDPPacket{server: tpc.server}}
	connect.SetSourceVirtualPortStreamID(tpc.LocalPort)
	connect.SetSourceVirtualPortStreamType(tpc.StreamType)
	connect.SetDestinationVirtualPortStreamID(tpc.RemotePort)
	connect.SetDestinationVirtualPortStreamType(tpc.StreamType)
	connect.SetType(constants.ConnectPacket)
	connect.AddFlag(constants.PacketFlagReliable)
	connect.AddFlag(constants.PacketFlagNeedsAck)
	connect.AddFlag(constants.PacketFlagHasSize)
	connect.setSignature(connect.calculateSignature([]byte{}, []byte{}))
	connectionSignature, _ := connect.calculateConnectionSignature(server.udpSocket.LocalAddr())
	connect.setConnectionSignature(connectionSignature)

	_, err := tpc.conn.Write(connect.Bytes())
	if err != nil {
		return err
	}

	buf := make([]byte, tpc.server.FragmentSize)
	tpc.conn.SetReadDeadline(time.Now().Add(time.Second))
	_, err = tpc.conn.Read(buf)
	if err != nil {
		return err
	}

	readStream := NewByteStreamIn(buf, server.LibraryVersions, server.ByteStreamSettings)
	ack, err := NewPRUDPPacketV0(server, nil, readStream)
	if err != nil {
		return err
	}

	if !(ack.HasFlag(constants.PacketFlagAck) && ack.Type() == constants.ConnectPacket && ack.sequenceID == connect.sequenceID) {
		return fmt.Errorf("Expected connect ack, got: %v", ack)
	}

	return nil
}

func (tpc *TestPRUDPClient) Stop() error {

	return nil
}

func (tpc *TestPRUDPClient) SendPayload(payload []byte) error {

	if tpc.State != StateConnected {
		return fmt.Errorf("Invalid state to send payload: %v", tpc.State)
	}

	var fragmentId uint8 = 1

	for len(payload) > 0 {

		if len(payload) <= tpc.server.FragmentSize {
			fragmentId = 0
		}

		tpc.sendFragment(payload[:tpc.server.FragmentSize], fragmentId)
		payload = payload[tpc.server.FragmentSize:]
	}

	return nil
}

func (tpc *TestPRUDPClient) sendFragment(fragment []byte, fragmentId uint8) error {

	data := PRUDPPacketV0{PRUDPPacket{server: server}}
	data.SetSourceVirtualPortStreamID(tpc.LocalPort)
	data.SetSourceVirtualPortStreamType(tpc.StreamType)
	data.SetDestinationVirtualPortStreamID(tpc.RemotePort)
	data.SetDestinationVirtualPortStreamType(tpc.StreamType)
	data.SetType(constants.DataPacket)
	data.AddFlag(constants.PacketFlagReliable)
	data.AddFlag(constants.PacketFlagNeedsAck)
	data.AddFlag(constants.PacketFlagHasSize)
	data.SetSequenceID(tpc.sequenceCounter.Next())
	data.SetPayload(fragment)
	data.setFragmentID(fragmentId)

	// TODO all of the signature stuff is wrong I think :D
	data.setSignature(data.calculateSignature([]byte{}, []byte{}))
	connectionSignature, _ := data.calculateConnectionSignature(server.udpSocket.LocalAddr())
	data.setConnectionSignature(connectionSignature)

	_, err := client.Write(data.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (tpc *TestPRUDPClient) ExpectPayload(buffer []byte) error {

	return nil
}

// Returns a new TestPRUDPClient
func NewPRUDPClient(server *PRUDPServer, streamType constants.StreamType, remotePort uint8) *TestPRUDPClient {

	return &TestPRUDPClient{
		server:          server,
		sequenceCounter: NewSequenceCounter(),
		State:           StateNotConnected,
		StreamType:      streamType,
		LocalPort:       1,
		RemotePort:      remotePort,
	}
}

type SequenceCounter struct {
	lock    sync.Mutex
	current uint16
}

// Returns a new SequenceCounter
func NewSequenceCounter() *SequenceCounter {
	return &SequenceCounter{
		current: 1,
	}
}

// Returns the next sequence id and updates the internal counter
func (sc *SequenceCounter) Next() uint16 {

	sc.lock.Lock()
	next := sc.current
	sc.current++
	sc.lock.Unlock()

	return next
}
