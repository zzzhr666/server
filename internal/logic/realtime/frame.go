package realtime

import (
	"encoding/binary"
	"hash/crc32"
	"io"
)

var checksumTable = crc32.MakeTable(crc32.Castagnoli)

const magicNumber = "GRTP"

const version byte = 1

const payloadMaxLength = 1 << 20

const headerLength = 13

func readFrame(reader io.Reader) ([]byte, error) {
	var header [headerLength]byte
	if _, err := io.ReadFull(reader, header[:]); err != nil {
		return nil, err
	}
	magic := header[:4]
	ver := header[4]
	length := binary.BigEndian.Uint32(header[5:9])
	checksum := binary.BigEndian.Uint32(header[9:13])
	if string(magic) != magicNumber {
		return nil, errInvalidFrameMagic
	}
	if ver != version {
		return nil, errUnsupportedFrameVersion
	}

	if length == 0 || length > payloadMaxLength {
		return nil, errInvalidFrameLength
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, err
	}
	expectedChecksum := crc32.Checksum(payload, checksumTable)
	if checksum != expectedChecksum {
		return nil, errFrameChecksumMismatch
	}
	return payload, nil
}

func writeFrame(writer io.Writer, payload []byte) error {
	payloadLen := len(payload)
	if payloadLen == 0 || payloadLen > payloadMaxLength {
		return errInvalidFrameLength
	}

	var header [headerLength]byte
	copy(header[:4], magicNumber)
	header[4] = version
	binary.BigEndian.PutUint32(header[5:9], uint32(payloadLen))
	checksum := crc32.Checksum(payload, checksumTable)
	binary.BigEndian.PutUint32(header[9:13], checksum)
	if err := writeAll(writer, header[:]); err != nil {
		return err
	}
	return writeAll(writer, payload)
}

func writeAll(writer io.Writer, payload []byte) error {
	for len(payload) > 0 {
		n, err := writer.Write(payload)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		payload = payload[n:]
	}
	return nil
}
