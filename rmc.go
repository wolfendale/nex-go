package nex

// RMCRequest represets a RMC request
type RMCRequest struct {
	protocolID uint8
	callID     uint32
	methodID   uint32
	parameters []byte
}

// GetProtocolID sets the RMC request protocolID
func (request *RMCRequest) GetProtocolID() uint8 {
	return request.protocolID
}

// GetCallID sets the RMC request callID
func (request *RMCRequest) GetCallID() uint32 {
	return request.callID
}

// GetMethodID sets the RMC request methodID
func (request *RMCRequest) GetMethodID() uint32 {
	return request.methodID
}

// GetParameters sets the RMC request parameters
func (request *RMCRequest) GetParameters() []byte {
	return request.parameters
}

// NewRMCRequest returns a new parsed RMCRequest
func NewRMCRequest(data []byte) RMCRequest {
	stream := NewStream(data)

	_ = stream.ReadU32LENext(1)[0]
	protocolID := stream.ReadByteNext() ^ 0x80
	callID := stream.ReadU32LENext(1)[0]
	methodID := stream.ReadU32LENext(1)[0]
	parameters := data[13:]

	request := RMCRequest{
		protocolID: protocolID,
		callID:     callID,
		methodID:   methodID,
		parameters: parameters,
	}

	return request
}

// RMCResponse represents a RMC response
type RMCResponse struct {
	protocolID uint8
	success    int
	callID     uint32
	methodID   uint32
	data       []byte
	errorCode  uint32
}

// SetSuccess sets the RMCResponse payload to an instance of RMCSuccess
func (response *RMCResponse) SetSuccess(methodID uint32, data []byte) {
	response.success = 1
	response.methodID = methodID
	response.data = data
}

// SetError sets the RMCResponse payload to an instance of RMCError
func (response *RMCResponse) SetError(errorCode uint32) {
	response.success = 0
	response.errorCode = errorCode
}

// Bytes converts a RMCResponse struct into a usable byte array
func (response *RMCResponse) Bytes() []byte {
	body := NewStream()

	body.Grow(2)
	body.WriteByteNext(byte(response.protocolID))
	body.WriteByteNext(byte(response.success))

	if response.success == 1 {
		body.Grow(8)
		body.WriteU32LENext([]uint32{response.callID})
		body.WriteU32LENext([]uint32{response.methodID | 0x8000})

		body.Grow(int64(len(response.data)))
		body.WriteBytesNext(response.data)
	} else {
		body.Grow(8)
		body.WriteU32LENext([]uint32{response.errorCode})
		body.WriteU32LENext([]uint32{response.callID})
	}

	data := NewStream()
	data.Grow(int64(4 + len(body.Bytes())))

	data.WriteNEXBufferNext(body.Bytes())

	return data.Bytes()
}

// NewRMCResponse returns a new RMCResponse
func NewRMCResponse(protocolID uint8, callID uint32) RMCResponse {
	response := RMCResponse{
		protocolID: protocolID,
		callID:     callID,
	}

	return response
}
