package realtime

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"testing"
)

func TestFrameRoundTripWritesExpectedHeader(t *testing.T) {
	payload := []byte("match_start")
	var buffer bytes.Buffer

	if err := writeFrame(&buffer, payload); err != nil {
		t.Fatalf("writeFrame() error = %v", err)
	}

	raw := buffer.Bytes()
	if got, want := len(raw), headerLength+len(payload); got != want {
		t.Fatalf("frame length = %d, want %d", got, want)
	}
	if got := string(raw[:4]); got != magicNumber {
		t.Fatalf("magic = %q, want %q", got, magicNumber)
	}
	if got := raw[4]; got != version {
		t.Fatalf("version = %d, want %d", got, version)
	}
	if got, want := binary.BigEndian.Uint32(raw[5:9]), uint32(len(payload)); got != want {
		t.Fatalf("payload length = %d, want %d", got, want)
	}
	if got, want := binary.BigEndian.Uint32(raw[9:13]), crc32.Checksum(payload, checksumTable); got != want {
		t.Fatalf("checksum = %d, want %d", got, want)
	}

	got, err := readFrame(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

func TestReadFrameRejectsInvalidHeader(t *testing.T) {
	payload := []byte("heartbeat")
	tests := []struct {
		name   string
		mutate func([]byte)
		want   error
	}{
		{
			name: "magic",
			mutate: func(frame []byte) {
				frame[0] = 'X'
			},
			want: errInvalidFrameMagic,
		},
		{
			name: "version",
			mutate: func(frame []byte) {
				frame[4] = version + 1
			},
			want: errUnsupportedFrameVersion,
		},
		{
			name: "zero payload length",
			mutate: func(frame []byte) {
				binary.BigEndian.PutUint32(frame[5:9], 0)
			},
			want: errInvalidFrameLength,
		},
		{
			name: "oversized payload length",
			mutate: func(frame []byte) {
				binary.BigEndian.PutUint32(frame[5:9], payloadMaxLength+1)
			},
			want: errInvalidFrameLength,
		},
		{
			name: "checksum",
			mutate: func(frame []byte) {
				frame[9] ^= 0xff
			},
			want: errFrameChecksumMismatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := encodedFrame(t, payload)
			tt.mutate(frame)

			_, err := readFrame(bytes.NewReader(frame))
			if !errors.Is(err, tt.want) {
				t.Fatalf("readFrame() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestReadFrameRejectsTruncatedInput(t *testing.T) {
	frame := encodedFrame(t, []byte("match_resume"))
	tests := []struct {
		name  string
		input []byte
	}{
		{name: "header", input: frame[:headerLength-1]},
		{name: "payload", input: frame[:len(frame)-1]},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readFrame(bytes.NewReader(tt.input))
			if !errors.Is(err, io.ErrUnexpectedEOF) {
				t.Fatalf("readFrame() error = %v, want %v", err, io.ErrUnexpectedEOF)
			}
		})
	}
}

func TestWriteFrameRejectsInvalidPayloadLength(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
	}{
		{name: "empty", payload: nil},
		{name: "oversized", payload: make([]byte, payloadMaxLength+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := writeFrame(io.Discard, tt.payload); !errors.Is(err, errInvalidFrameLength) {
				t.Fatalf("writeFrame() error = %v, want %v", err, errInvalidFrameLength)
			}
		})
	}
}

func TestWriteFrameHandlesShortWrites(t *testing.T) {
	payload := []byte("match_cancel")
	writer := &limitedWriter{limit: 2}

	if err := writeFrame(writer, payload); err != nil {
		t.Fatalf("writeFrame() error = %v", err)
	}
	got, err := readFrame(bytes.NewReader(writer.Bytes()))
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

func TestWriteAllRejectsWriterWithoutProgress(t *testing.T) {
	if err := writeAll(noProgressWriter{}, []byte("payload")); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("writeAll() error = %v, want %v", err, io.ErrShortWrite)
	}
}

func encodedFrame(t *testing.T, payload []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	if err := writeFrame(&buffer, payload); err != nil {
		t.Fatalf("writeFrame() error = %v", err)
	}
	return buffer.Bytes()
}

type limitedWriter struct {
	bytes.Buffer
	limit int
}

func (w *limitedWriter) Write(payload []byte) (int, error) {
	if len(payload) > w.limit {
		payload = payload[:w.limit]
	}
	return w.Buffer.Write(payload)
}

type noProgressWriter struct{}

func (noProgressWriter) Write([]byte) (int, error) {
	return 0, nil
}
