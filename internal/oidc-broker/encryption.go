package oidc_broker

import (
	"encoding/base64"
	"errors"

	"github.com/gtank/cryptopasta"
)

type OidcEncryption struct {
	EncKey [32]byte
}

// EncryptState encrypts the state using a given key.
func EncryptState(state string, key string) (string, error) {
	byteKey, err := parseString(key)
	if err != nil {
		logger.Errorf("encrypt state: failed parsing: %s", err)
		return "", err
	}
	encrypted, err := cryptopasta.Encrypt([]byte(state), &byteKey)
	if err != nil {
		logger.Errorf("encrypt state: failed encrypting: %s", err)
		return "", err
	}
	return base64.URLEncoding.EncodeToString(encrypted), nil
}

// DecryptState decrypts the state using a given key.
func DecryptState(encryptedState string, key string) (string, error) {
	encrypted, err := base64.URLEncoding.DecodeString(encryptedState)
	if err != nil {
		logger.Errorf("failed decoding state: %s", err)
		return "", errors.New("failed decoding state")
	}
	byteKey, err := parseString(key)
	if err != nil {
		logger.Errorf("decrypt state: failed parsing: %s", err)
		return "", err
	}
	decrypted, err := cryptopasta.Decrypt(encrypted, &byteKey)
	if err != nil {
		logger.Errorf("failed decrypting state: %s", err)
		return "", errors.New("failed decrypting state")
	}
	return string(decrypted), nil
}

func parseString(key string) ([32]byte, error) {
	keySlice := []byte(key)

	if len(keySlice) != 32 {
		logger.Error("OIDC state key length is not 32 bytes")
		return [32]byte{}, errors.New("OIDC state key length is not 32 bytes")
	}

	keyByteArray := [32]byte{}
	copy(keyByteArray[:], keySlice)

	return keyByteArray, nil
}
