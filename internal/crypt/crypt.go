package crypt

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

const (
	magicHeader = "SCPASS1\x00"
	fileVersion = uint16(1)
)

var (
	errInvalidHeader = errors.New("invalid vault header")
	errInvalidVersion = errors.New("unsupported vault version")
)

type Params struct {
	Time    uint32
	Memory  uint32
	Threads uint8
	KeyLen  uint32
}

func DefaultParams() Params {
	return Params{
		Time:    3,
		Memory: 64 * 1024,
		Threads: 2,
		KeyLen:  32,
	}
}

func Encrypt(password string, plaintext []byte, params Params) ([]byte, error) {
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	key := argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Threads, params.KeyLen)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := aead.Seal(nil, nonce, plaintext, nil)

	buf := &bytes.Buffer{}
	buf.WriteString(magicHeader)
	if err := binary.Write(buf, binary.BigEndian, fileVersion); err != nil {
		return nil, err
	}
	if err := writeParams(buf, params); err != nil {
		return nil, err
	}
	if _, err := buf.Write(salt); err != nil {
		return nil, err
	}
	if _, err := buf.Write(nonce); err != nil {
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, uint32(len(ciphertext))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(ciphertext); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func Decrypt(password string, data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)

	header := make([]byte, len(magicHeader))
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}
	if string(header) != magicHeader {
		return nil, errInvalidHeader
	}

	var version uint16
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		return nil, err
	}
	if version != fileVersion {
		return nil, errInvalidVersion
	}

	params, err := readParams(reader)
	if err != nil {
		return nil, err
	}

	salt := make([]byte, 16)
	if _, err := io.ReadFull(reader, salt); err != nil {
		return nil, err
	}
	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(reader, nonce); err != nil {
		return nil, err
	}

	var cipherLen uint32
	if err := binary.Read(reader, binary.BigEndian, &cipherLen); err != nil {
		return nil, err
	}
	ciphertext := make([]byte, cipherLen)
	if _, err := io.ReadFull(reader, ciphertext); err != nil {
		return nil, err
	}

	key := argon2.IDKey([]byte(password), salt, params.Time, params.Memory, params.Threads, params.KeyLen)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	return aead.Open(nil, nonce, ciphertext, nil)
}

func writeParams(w io.Writer, params Params) error {
	if err := binary.Write(w, binary.BigEndian, params.Time); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, params.Memory); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, params.Threads); err != nil {
		return err
	}
	return binary.Write(w, binary.BigEndian, params.KeyLen)
}

func readParams(r io.Reader) (Params, error) {
	var params Params
	if err := binary.Read(r, binary.BigEndian, &params.Time); err != nil {
		return params, err
	}
	if err := binary.Read(r, binary.BigEndian, &params.Memory); err != nil {
		return params, err
	}
	if err := binary.Read(r, binary.BigEndian, &params.Threads); err != nil {
		return params, err
	}
	if err := binary.Read(r, binary.BigEndian, &params.KeyLen); err != nil {
		return params, err
	}
	return params, nil
}
