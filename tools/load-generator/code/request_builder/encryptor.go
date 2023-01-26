package requestbuilder

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
)

const (
	defaultAESKey  = "0123456789ABCDEF"
	defaultKeySize = 16
	uint64Bytes    = 8
)

// Key is an alias to shorten syntax
type Key = []byte

// Encryptor encrypts and decrypts an uint64 id into `size` bytes AES
type Encryptor struct {
	cipher cipher.Block
	size   int

	zeroBuffer      []byte
	textBuffer      []byte
	encryptedBuffer []byte
}

// NewDefaultEncryptor create a new instance of Encryptor
// with default key (`[]byte("0123456789ABCDEF")`) and key size (16)
func NewDefaultEncryptor() (*Encryptor, error) {
	key := []byte(defaultAESKey)
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	size := defaultKeySize

	return &Encryptor{
		cipher:          c,
		size:            size,
		zeroBuffer:      make([]byte, size),
		textBuffer:      make([]byte, size),
		encryptedBuffer: make([]byte, size),
	}, nil
}

// Encrypt converts `uint64` value into byte slice and encrypts using AES.
// Because of heap allocation optimization, result is returned in
// underlying buffer.
func (e *Encryptor) Encrypt(id uint64) Key {
	copy(e.textBuffer, e.zeroBuffer)
	copy(e.encryptedBuffer, e.zeroBuffer)

	binary.BigEndian.PutUint64(e.textBuffer[(e.size-uint64Bytes):], id)
	e.cipher.Encrypt(e.encryptedBuffer, e.textBuffer)
	return e.encryptedBuffer
}

// Decrypt decrypts a value using AES and converts it to `uint64`
func (e *Encryptor) Decrypt(encrypted Key) uint64 {
	bytes := make(Key, e.size)
	e.cipher.Decrypt(bytes, encrypted)
	return binary.BigEndian.Uint64(bytes[(e.size - uint64Bytes):])
}
