package nex

// PacketInterface defines all the methods a packet for both PRUDP and HPP should have
type PacketInterface interface {
	Sender() ClientInterface
	Payload() []byte
	SetPayload(payload []byte)
	RMCMessage() *RMCMessage
	SetRMCMessage(message *RMCMessage)
}
