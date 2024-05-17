package nex

import (
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/PretendoNetwork/nex-go/v2/constants"
	"github.com/stretchr/testify/assert"
)

var server *PRUDPServer
var client *net.UDPConn

func TestSynAck(t *testing.T) {

	endpoint := NewPRUDPEndPoint(1)
	server.BindPRUDPEndPoint(endpoint)

	endpoint.OnData(func(packet PacketInterface) {
		assert.Equal(t, "foobar", string(packet.Payload()))
	})

	sendSyn(t)
	sendConnect(t)
	sendData(t)
}

func sendSyn(t *testing.T) {

	// Syn
	syn := PRUDPPacketV0{PRUDPPacket{server: server}}
	syn.SetSourceVirtualPortStreamID(1)
	syn.SetSourceVirtualPortStreamType(constants.StreamTypeDO)
	syn.SetDestinationVirtualPortStreamID(1)
	syn.SetDestinationVirtualPortStreamType(constants.StreamTypeDO)
	syn.SetType(constants.SynPacket)
	syn.AddFlag(constants.PacketFlagNeedsAck)
	syn.setSignature(syn.calculateSignature([]byte{}, []byte{}))
	connectionSignature, _ := syn.calculateConnectionSignature(server.udpSocket.LocalAddr())
	syn.setConnectionSignature(connectionSignature)

	client.SetDeadline(time.Now().Add(time.Second))
	write, err := client.Write(syn.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, 16, write)

	// Ack
	buf := make([]byte, 962)
	client.SetDeadline(time.Now().Add(time.Second))
	read, err := client.Read(buf)

	assert.NoError(t, err)
	assert.Equal(t, 18, read)

	readStream := NewByteStreamIn(buf, server.LibraryVersions, server.ByteStreamSettings)
	synAck, err := NewPRUDPPacketV0(server, nil, readStream)

	assert.NoError(t, err)
	assert.Equal(t, constants.SynPacket, synAck.Type())
	assert.Equal(t, constants.PacketFlagAck, constants.PacketFlagAck&synAck.Flags())
	assert.Equal(t, uint16(0), synAck.SequenceID())
}

func sendConnect(t *testing.T) {

	connect := PRUDPPacketV0{PRUDPPacket{server: server}}
	connect.SetSourceVirtualPortStreamID(1)
	connect.SetSourceVirtualPortStreamType(constants.StreamTypeDO)
	connect.SetDestinationVirtualPortStreamID(1)
	connect.SetDestinationVirtualPortStreamType(constants.StreamTypeDO)
	connect.SetType(constants.ConnectPacket)
	connect.AddFlag(constants.PacketFlagReliable)
	connect.AddFlag(constants.PacketFlagNeedsAck)
	connect.AddFlag(constants.PacketFlagHasSize)
	connect.setSignature(connect.calculateSignature([]byte{}, []byte{}))
	connectionSignature, _ := connect.calculateConnectionSignature(server.udpSocket.LocalAddr())
	connect.setConnectionSignature(connectionSignature)

	client.SetDeadline(time.Now().Add(time.Second))
	write, err := client.Write(connect.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, 18, write)

	// Ack
	buf := make([]byte, 962)
	client.SetDeadline(time.Now().Add(time.Second))
	read, err := client.Read(buf)

	assert.NoError(t, err)
	assert.Equal(t, 18, read)

	readStream := NewByteStreamIn(buf, server.LibraryVersions, server.ByteStreamSettings)
	connectAck, err := NewPRUDPPacketV0(server, nil, readStream)

	assert.NoError(t, err)
	assert.Equal(t, constants.ConnectPacket, connectAck.Type())
	assert.Equal(t, constants.PacketFlagAck, constants.PacketFlagAck&connectAck.Flags())
	assert.Equal(t, uint16(1), connectAck.SequenceID())
}

func sendData(t *testing.T) {

	payload := []byte("foobar")

	data := PRUDPPacketV0{PRUDPPacket{server: server}}
	data.SetSourceVirtualPortStreamID(1)
	data.SetSourceVirtualPortStreamType(constants.StreamTypeDO)
	data.SetDestinationVirtualPortStreamID(1)
	data.SetDestinationVirtualPortStreamType(constants.StreamTypeDO)
	data.SetType(constants.DataPacket)
	data.AddFlag(constants.PacketFlagReliable)
	data.AddFlag(constants.PacketFlagNeedsAck)
	data.AddFlag(constants.PacketFlagHasSize)
	data.SetSequenceID(2)
	data.SetPayload(payload)
	data.setSignature(data.calculateSignature([]byte{}, []byte{}))
	connectionSignature, _ := data.calculateConnectionSignature(server.udpSocket.LocalAddr())
	data.setConnectionSignature(connectionSignature)

	client.SetDeadline(time.Now().Add(time.Second))
	write, err := client.Write(data.Bytes())
	assert.NoError(t, err)
	assert.Equal(t, 21, write)

	time.Sleep(time.Second)
}

func TestMain(m *testing.M) {

	go startServer()
	time.Sleep(1 * time.Second)
	startClient()

	code := m.Run()

	os.Exit(code)
}

func startServer() {

	server = NewPRUDPServer()
	server.LibraryVersions.SetDefault(NewLibraryVersion(1, 1, 0))
	server.AccessKey = "ridfebb9"
	server.Listen(0)
}

func startClient() {

	addr := server.udpSocket.LocalAddr().(*net.UDPAddr)

	var err error
	client, err = net.DialUDP("udp", nil, addr)

	if err != nil {
		fmt.Printf("Error trying to start UDP client: %v\n", err)
		os.Exit(1)
	}
}
