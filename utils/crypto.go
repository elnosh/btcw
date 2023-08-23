package utils

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/nacl/secretbox"
)

type params struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
	saltlen     uint32
	keylen      uint32
}

func HashPassphrase(passphrase []byte) (string, error) {
	p := &params{
		memory:      64 * 1024,
		iterations:  3,
		parallelism: 2,
		saltlen:     16,
		keylen:      32,
	}

	// generate random salt
	salt := make([]byte, p.saltlen)
	_, err := rand.Read(salt)
	if err != nil {
		return "", err
	}

	key := argon2.IDKey(passphrase, salt, p.iterations, p.memory, p.parallelism, p.keylen)
	b64salt := base64.RawStdEncoding.EncodeToString(salt)
	b64key := base64.RawStdEncoding.EncodeToString(key)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, p.memory, p.iterations, p.parallelism, b64salt, b64key)
	return encoded, nil
}

func VerifyPassphrase(encodedHash, passphrase string) bool {
	p, key, salt, err := DecodeHash(encodedHash)
	if err != nil {
		return false
	}

	newkey := argon2.IDKey([]byte(passphrase), salt, p.iterations, p.memory, p.parallelism, p.keylen)
	return subtle.ConstantTimeCompare(key, newkey) == 1
}

func DecodeHash(encodedHash string) (p *params, key, salt []byte, err error) {
	invalidHashErr := errors.New("invalid hash")
	split := strings.Split(encodedHash, "$")
	if len(split) != 6 {
		return nil, nil, nil, invalidHashErr
	}

	var version int
	_, err = fmt.Sscanf(split[2], "v=%d", &version)
	if err != nil {
		return nil, nil, nil, invalidHashErr
	}
	if version != argon2.Version {
		return nil, nil, nil, invalidHashErr
	}

	p = &params{}
	_, err = fmt.Sscanf(split[3], "m=%d,t=%d,p=%d", &p.memory, &p.iterations, &p.parallelism)
	if err != nil {
		return nil, nil, nil, invalidHashErr
	}

	salt, err = base64.RawStdEncoding.Strict().DecodeString(split[4])
	if err != nil {
		return nil, nil, nil, invalidHashErr
	}
	key, err = base64.RawStdEncoding.Strict().DecodeString(split[5])
	if err != nil {
		return nil, nil, nil, invalidHashErr
	}
	p.keylen = uint32(len(key))

	return p, key, salt, nil
}

func Encrypt(input, key []byte) ([]byte, error) {
	var nonce [24]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		return nil, err
	}

	var secret [32]byte
	copy(secret[:], key)

	encrypted := secretbox.Seal(nonce[:], input, &nonce, &secret)
	return encrypted, nil
}

func Decrypt(input, key []byte) ([]byte, error) {
	var nonce [24]byte
	copy(nonce[:], input[:24])

	var secret [32]byte
	copy(secret[:], key)

	decrypted, ok := secretbox.Open(nil, input[24:], &nonce, &secret)
	if !ok {
		return nil, errors.New("decryption error")
	}

	return decrypted, nil
}
