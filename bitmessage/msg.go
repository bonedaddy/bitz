// Package bitmessage implements the BitMessage protocol.
package bitmessage

// The description of most data types was based on the protocol definition,
// licensed under "Creative Commons Attribution 3.0" and available at
// https://bitmessage.org/wiki/Protocol_specification.

import (
	"crypto/sha512"

	"code.google.com/p/go.crypto/ripemd160"
)

// Magic value indicating message origin network, and used to seek to next
// message when stream state is unknown.
var MagicHeader = []byte("\xE9\xBE\xB4\xD9")

type Message struct {
	// Magic value indicating message origin network, and used to seek to
	// next message when stream state is unknown.
	magic uint32
	// ASCII string identifying the packet content, NULL padded (non-NULL
	// padding results in packet rejected).
	command [12]byte
	// Length of payload in number of bytes
	length uint32
	// First 4 bytes of sha512(payload)
	checksum uint32
	// The actual data, a message or an object
	payload []byte
}

// NetworkAddress identifies an address in the BitMessage network. Network
// addresses are not prefixed with a timestamp in the version message.
type NetworkAddress struct {
	time     uint32 // the time
	stream   uint32 // Stream number for this node
	services uint64 // Same service(s) listed in version

	// IPv6 Address, or IPv6-mapped IPv4 address:
	//00 00 00 00 00 00 00 00 00 00 FF FF, followed by the IPv4 bytes.
	ip   [16]byte
	port uint16 // portNumber.
}

// InventoryVectors are used for notifying other nodes about objects they
// have or data which is being requested. Two rounds of SHA-512 are used,
// resulting in a 64 byte hash. Only the first 32 bytes are used; the later 32
// bytes are ignored.
type InventoryVector struct {
	hash [32]byte
}

// Use varint and varstring from:
// https://github.com/spearson78/guardian/tree/master/encoding
type varint []byte
type varstring []byte

type UnencryptedMessageData struct {
	// Message format version.
	msgVersion varint
	// Sender's address version number. This is needed in order to calculate
	// the sender's address to show in the UI, and also to allow for forwards
	// compatible changes to the public-key data included below.
	addressVersion varint
	// Sender's stream number.
	streamNumber varint
	// A bitfield of optional behaviors and features that can be expected from
	// the node with this pubkey included in this msg message (the sender's
	// pubkey).
	behavior uint32
	// The ECC public key used for signing (uncompressed format; normally
	// prepended with \x04).
	publicSigningKey [64]byte
	// The ECC public key used for encryption (uncompressed format; normally
	// prepended with \x04 ).
	publicEncryptionKey [64]byte
	// The ripe hash of the public key of the receiver of the message.
	destinationRipe [20]byte
	// Message encoding type.
	encoding varint
	// Message length.
	messageLength varint
	// The message.
	message []byte
	// Length of the acknowledgement data
	ackLength varint
	// The acknowledgement data to be transmitted. This takes the form of a
	// Bitmessage protocol message, like another msg message. The POW therein
	// must already be completed.
	ackData []byte
	// Length of the signature.
	sigLength varint
	// The ECDSA signature which covers everything from the msg_version to the
	// ack_data.
	singnature []byte
}

const (
	// Any data with this number may be ignored. The sending node might simply
	// be sharing its public key with you.
	EncodingIgnore = iota
	// UTF-8. No 'Subject' or 'Body' sections. Useful for simple strings of
	// data, like URIs or magnet links.
	EncodingTrivial
	// UTF-8. Uses 'Subject' and 'Body' sections. No MIME is used.
	// messageToTransmit = 'Subject:' + subject + '\n' + 'Body:' + message
	EncodingSimple
	// Further values for the message encodings can be decided upon by the
	// community. Any MIME or MIME-like encoding format, should they be used,
	// should make use of Bitmessage's 8-bit bytes.
	// As of 2013-04-14, no other types have been standardized.
)

const (
	// Pubkey bitfield features. As of 2013-04-14, only the following is in
	// the protocol:
	// If true, the receiving node does send acknowledgements (rather than
	// dropping them). Note that this is the least significant bit.
	pubKeyDoesAck = 31
)

// When a node creates an outgoing connection, it will immediately advertise
// its version. The remote node will respond with its version. No futher
// communication is possible until both peers have exchanged their version.
type VersionMessage struct {
	// Identifies protocol version being used by the node.
	version int32
	// bitfield of features to be enabled for this connection.
	services uint64
	// standard UNIX timestamp in seconds
	timestamp int64
	// The network address of the node receiving this message (not including
	// the time or stream number)
	addrRecv NetworkAddress
	// The network address of the node emitting this message (not including
	// the time or stream number and the ip itself is ignored by the receiver)
	addrFrom NetworkAddress
	// Random nonce used to detect connections to self.
	nonce uint64
	// User Agent (0x00 if string is 0 bytes long)
	userAgent varstring
	// The stream numbers that the emitting node is interested in.
	streamNumbers []varint
}

const (
	// This is a normal network node.
	ConnectionServiceNodeNetwork = 1
)

// The VerackMessage is sent in reply to version. This message consists of
// only a message header with the command string "verack".
type VerackMessage struct {
	msg Message // Contains only a header with the command string "verack"
}

// Provide information on known nodes of the network. Non-advertised nodes
// should be forgotten after typically 3 hours.
type AddrMessage struct {
	count    varint           // Number of address entries (max: 1000)
	addrList []NetworkAddress // Address of other nodes on the network.
}

// InvMessage allows a node to advertise its knowledge of one or more objects.
// It can be received unsolicited, or in reply to getmessages.
// Maximum payload length: 50000 items.
type InvMessage struct {
	count     varint
	inventory []InventoryVector // max 50000 items.
}

// Objects:
// Any object is also a message. The difference is, that an object should be
// shared with the whole stream, while a normal message is just for direct
// client to client communication. A client should advertise objects that are
// not older than 2 days. To create an object, the Proof Of Work has to be
// done.

// When a node has the hash of a public key (from an address) but not the
// public key itself, it must send out a request for the public key.
type GetPubKey struct {
	powNonce       uint64   // Random nonce used for the Proof Of Work
	time           uint32   // The time that this message was generated and broadcast.
	addressVersion varint   // The address' version.
	streamNumber   varint   // The address' stream number
	pubKeyHash     [20]byte // The ripemd hash of the public key
}

// A public key.
type PubKey struct {
	powNonce       uint64 // Random nonce used for the Proof Of Work
	time           uint32 // The time that this message was generated and broadcast.
	addressVersion varint // The address' version.
	streamNumber   varint // The address' stream number
	behavior       uint32 // A bitfield of optional behaviors and features that can be expected from the node receiving the message.
	// The ECC public key used for signing (uncompressed format; normally
	// prepended with \x04).
	publicSigningKey [64]byte
	// The ECC public key used for encryption (uncompressed format; normally
	// prepended with \x04 ).
	publicEncryptionKey [64]byte
}

// Used for person-to-person messages.
type Msg struct {
	powNonce       uint64 // Random nonce used for the Proof Of Work
	time           uint32 // The time that this message was generated and broadcast.
	addressVersion varint // The address' version.
	streamNumber   varint // The address' stream number
	encrypted      []byte // Encrypted data. See also: UnencryptedMessageData
}

type Broadcast struct {
	// Random nonce used for the Proof Of Work
	powNonce uint64
	// The time that this message was generated and broadcast.
	time uint32
	// The version number of this broadcast protocol message.
	broadcastVersion varint
	// Sender's address version number. This is needed in order to calculate
	// the sender's address to show in the UI, and also to allow for forwards
	// compatible changes to the public-key data included below.
	addressVersion varint
	// Sender's stream number.
	streamNumber varint
	// A bitfield of optional behaviors and features that can be expected from
	// the node with this pubkey included in this msg message (the sender's
	// pubkey).
	behavior uint32
	// The ECC public key used for signing (uncompressed format; normally
	// prepended with \x04).
	publicSigningKey [64]byte
	// The ECC public key used for encryption (uncompressed format; normally
	// prepended with \x04 ).
	publicEncryptionKey [64]byte
	// The sender's address hash. This is included so that nodes can more
	// cheaply detect whether this is a broadcast message for which they are
	// listening, although it must be verified with the public key above.
	addressHash [20]byte
	// Message encoding type.
	encoding varint
	// Message length.
	messageLength varint
	// The message.
	message []byte
	// Length of the signature.
	sigLength varint
	// The ECDSA signature which covers everything from the msg_version to the
	// ack_data.
	singnature []byte
}

func ProofOfWork(msg []byte) ([]byte, error) {
	for i := 0; i < 2; i++ {
		h := sha512.New()
		h.Write(msg)
		msg = h.Sum(nil)
	}
	return msg, nil
}

// Bitmessage produces a hash for the provided message using a SHA-512 in the
// first round and a RIPEMD-160 in the second.
func Bitmessage(msg []byte) ([]byte, error) {
	s := sha512.New()
	s.Write(msg)

	r := ripemd160.New()
	r.Write(s.Sum(nil))
	return r.Sum(nil), nil
}
