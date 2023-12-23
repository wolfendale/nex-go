package nex

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/rand"
	"crypto/rc4"
	"errors"
	"fmt"

	"github.com/PretendoNetwork/nex-go/types"
)

// KerberosEncryption is a struct representing a Kerberos encryption utility
type KerberosEncryption struct {
	key []byte
}

// Validate checks the integrity of the given buffer by verifying the HMAC checksum
func (ke *KerberosEncryption) Validate(buffer []byte) bool {
	data := buffer[:len(buffer)-0x10]
	checksum := buffer[len(buffer)-0x10:]
	mac := hmac.New(md5.New, ke.key)

	mac.Write(data)

	return hmac.Equal(checksum, mac.Sum(nil))
}

// Decrypt decrypts the provided buffer if it passes the integrity check
func (ke *KerberosEncryption) Decrypt(buffer []byte) ([]byte, error) {
	if !ke.Validate(buffer) {
		return nil, errors.New("Invalid Kerberos checksum (incorrect password)")
	}

	cipher, err := rc4.NewCipher(ke.key)
	if err != nil {
		return nil, err
	}

	decrypted := make([]byte, len(buffer)-0x10)

	cipher.XORKeyStream(decrypted, buffer[:len(buffer)-0x10])

	return decrypted, nil
}

// Encrypt encrypts the given buffer and appends an HMAC checksum for integrity
func (ke *KerberosEncryption) Encrypt(buffer []byte) []byte {
	cipher, _ := rc4.NewCipher(ke.key)
	encrypted := make([]byte, len(buffer))

	cipher.XORKeyStream(encrypted, buffer)

	mac := hmac.New(md5.New, ke.key)

	mac.Write(encrypted)

	checksum := mac.Sum(nil)

	return append(encrypted, checksum...)
}

// NewKerberosEncryption creates a new KerberosEncryption instance with the given key.
func NewKerberosEncryption(key []byte) *KerberosEncryption {
	return &KerberosEncryption{key: key}
}

// KerberosTicket represents a ticket granting a user access to a secure server
type KerberosTicket struct {
	SessionKey   []byte
	TargetPID    *types.PID
	InternalData *types.Buffer
}

// Encrypt writes the ticket data to the provided stream and returns the encrypted byte slice
func (kt *KerberosTicket) Encrypt(key []byte, stream *StreamOut) ([]byte, error) {
	encryption := NewKerberosEncryption(key)

	stream.Grow(int64(len(kt.SessionKey)))
	stream.WriteBytesNext(kt.SessionKey)

	kt.TargetPID.WriteTo(stream)
	kt.InternalData.WriteTo(stream)

	return encryption.Encrypt(stream.Bytes()), nil
}

// NewKerberosTicket returns a new Ticket instance
func NewKerberosTicket() *KerberosTicket {
	return &KerberosTicket{}
}

// KerberosTicketInternalData holds the internal data for a kerberos ticket to be processed by the server
type KerberosTicketInternalData struct {
	Issued     *types.DateTime
	SourcePID  *types.PID
	SessionKey []byte
}

// Encrypt writes the ticket data to the provided stream and returns the encrypted byte slice
func (ti *KerberosTicketInternalData) Encrypt(key []byte, stream *StreamOut) ([]byte, error) {
	ti.Issued.WriteTo(stream)
	ti.SourcePID.WriteTo(stream)

	stream.Grow(int64(len(ti.SessionKey)))
	stream.WriteBytesNext(ti.SessionKey)

	data := stream.Bytes()

	if stream.Server.(*PRUDPServer).kerberosTicketVersion == 1 {
		ticketKey := make([]byte, 16)
		_, err := rand.Read(ticketKey)
		if err != nil {
			return nil, fmt.Errorf("Failed to generate ticket key. %s", err.Error())
		}

		hash := md5.Sum(append(key, ticketKey...))
		finalKey := hash[:]

		encryption := NewKerberosEncryption(finalKey)

		encrypted := encryption.Encrypt(data)

		finalStream := NewStreamOut(stream.Server)

		var ticketBuffer types.Buffer = ticketKey
		var encryptedBuffer types.Buffer = encrypted

		ticketBuffer.WriteTo(finalStream)
		encryptedBuffer.WriteTo(finalStream)

		return finalStream.Bytes(), nil
	}

	encryption := NewKerberosEncryption([]byte(key))

	return encryption.Encrypt(data), nil
}

// Decrypt decrypts the given data and populates the struct
func (ti *KerberosTicketInternalData) Decrypt(stream *StreamIn, key []byte) error {
	if stream.Server.(*PRUDPServer).kerberosTicketVersion == 1 {
		ticketKey := types.NewBuffer([]byte{})
		if err := ticketKey.ExtractFrom(stream); err != nil {
			return fmt.Errorf("Failed to read Kerberos ticket internal data key. %s", err.Error())
		}

		data := types.NewBuffer([]byte{})
		if err := ticketKey.ExtractFrom(stream); err != nil {
			return fmt.Errorf("Failed to read Kerberos ticket internal data. %s", err.Error())
		}

		hash := md5.Sum(append(key, *ticketKey...))
		key = hash[:]

		stream = NewStreamIn(*data, stream.Server)
	}

	encryption := NewKerberosEncryption(key)

	decrypted, err := encryption.Decrypt(stream.Bytes())
	if err != nil {
		return fmt.Errorf("Failed to decrypt Kerberos ticket internal data. %s", err.Error())
	}

	stream = NewStreamIn(decrypted, stream.Server)

	timestamp := types.NewDateTime(0)
	if err := timestamp.ExtractFrom(stream); err != nil {
		return fmt.Errorf("Failed to read Kerberos ticket internal data timestamp %s", err.Error())
	}

	userPID := types.NewPID(0)
	if err := userPID.ExtractFrom(stream); err != nil {
		return fmt.Errorf("Failed to read Kerberos ticket internal data user PID %s", err.Error())
	}

	ti.Issued = timestamp
	ti.SourcePID = userPID
	ti.SessionKey = stream.ReadBytesNext(int64(stream.Server.(*PRUDPServer).kerberosKeySize))

	return nil
}

// NewKerberosTicketInternalData returns a new KerberosTicketInternalData instance
func NewKerberosTicketInternalData() *KerberosTicketInternalData {
	return &KerberosTicketInternalData{}
}

// DeriveKerberosKey derives a users kerberos encryption key based on their PID and password
func DeriveKerberosKey(pid *types.PID, password []byte) []byte {
	key := password

	for i := 0; i < 65000+int(pid.Value())%1024; i++ {
		hash := md5.Sum(key)
		key = hash[:]
	}

	return key
}
