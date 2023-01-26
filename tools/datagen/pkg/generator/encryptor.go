package generator

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

// Encryptor encrypts and decrypts an uint64 id into `size` bytes AES
type Encryptor struct {
	key    []byte
	cipher cipher.Block
	size   int
}

// NewDefaultEncryptor create a new instance of Encryptor
// with default key (`[]byte("0123456789ABCDEF")`) and key size (16)
func NewDefaultEncryptor() (*Encryptor, error) {
	key := []byte(defaultAESKey)
	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	return &Encryptor{key: key, cipher: c, size: defaultKeySize}, nil
}

// Encrypt converts `uint64` value into byte slice and encrypts using AES
func (e *Encryptor) Encrypt(id uint64) Key {
	bytes := make(Key, e.size)
	encrypted := make(Key, e.size)
	binary.BigEndian.PutUint64(bytes[(e.size-uint64Bytes):], id)
	e.cipher.Encrypt(encrypted, bytes)
	return encrypted
}

// Decrypt decrypts a value using AES and converts it to `uint64`
func (e *Encryptor) Decrypt(encrypted Key) uint64 {
	bytes := make(Key, e.size)
	e.cipher.Decrypt(bytes, encrypted)
	return binary.BigEndian.Uint64(bytes[(e.size - uint64Bytes):])
}
