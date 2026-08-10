package realtime

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"server/internal/contract/realtimepb"

	"google.golang.org/protobuf/proto"
)

func TestReadClientEnvelope(t *testing.T) {
	want := &realtimepb.ClientEnvelope{
		RequestId: 42,
		Payload: &realtimepb.ClientEnvelope_Heartbeat{
			Heartbeat: &realtimepb.HeartbeatRequest{},
		},
	}

	got, err := readClientEnvelope(bytes.NewReader(encodedClientEnvelope(t, want)))
	if err != nil {
		t.Fatalf("readClientEnvelope() error = %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("client envelope = %v, want %v", got, want)
	}
}

func TestReadClientEnvelopeRejectsInvalidEnvelope(t *testing.T) {
	tests := []struct {
		name  string
		input func(t *testing.T) []byte
		want  error
	}{
		{
			name: "zero request ID",
			input: func(t *testing.T) []byte {
				return encodedClientEnvelope(t, &realtimepb.ClientEnvelope{
					Payload: &realtimepb.ClientEnvelope_Heartbeat{
						Heartbeat: &realtimepb.HeartbeatRequest{},
					},
				})
			},
			want: errInvalidRequestID,
		},
		{
			name: "empty payload",
			input: func(t *testing.T) []byte {
				return encodedClientEnvelope(t, &realtimepb.ClientEnvelope{RequestId: 1})
			},
			want: errInvalidPayload,
		},
		{
			name: "invalid protobuf",
			input: func(t *testing.T) []byte {
				return encodedFrame(t, []byte{0xff})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readClientEnvelope(bytes.NewReader(tt.input(t)))
			if tt.want != nil {
				if !errors.Is(err, tt.want) {
					t.Fatalf("readClientEnvelope() error = %v, want %v", err, tt.want)
				}
				return
			}
			if err == nil {
				t.Fatal("readClientEnvelope() error = nil, want protobuf decode error")
			}
		})
	}
}

func TestWriteServerEnvelope(t *testing.T) {
	want := &realtimepb.ServerEnvelope{
		RequestId: 42,
		Payload: &realtimepb.ServerEnvelope_MatchResult{
			MatchResult: &realtimepb.MatchResult{
				Status:         "matched",
				RoomName:       "room-7-8",
				Token:          "match-token",
				BattleNodeName: "battle-1",
				BattleUdpAddr:  "127.0.0.1:7001",
			},
		},
	}
	var buffer bytes.Buffer

	if err := writeServerEnvelope(&buffer, want); err != nil {
		t.Fatalf("writeServerEnvelope() error = %v", err)
	}
	payload, err := readFrame(bytes.NewReader(buffer.Bytes()))
	if err != nil {
		t.Fatalf("readFrame() error = %v", err)
	}
	var got realtimepb.ServerEnvelope
	if err := proto.Unmarshal(payload, &got); err != nil {
		t.Fatalf("proto.Unmarshal() error = %v", err)
	}
	if !proto.Equal(&got, want) {
		t.Fatalf("server envelope = %v, want %v", &got, want)
	}
}

func TestWriteServerEnvelopeRejectsEmptyPayload(t *testing.T) {
	tests := []struct {
		name     string
		envelope *realtimepb.ServerEnvelope
	}{
		{name: "nil"},
		{name: "empty", envelope: &realtimepb.ServerEnvelope{RequestId: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := writeServerEnvelope(io.Discard, tt.envelope); !errors.Is(err, errInvalidPayload) {
				t.Fatalf("writeServerEnvelope() error = %v, want %v", err, errInvalidPayload)
			}
		})
	}
}

func encodedClientEnvelope(t *testing.T, envelope *realtimepb.ClientEnvelope) []byte {
	t.Helper()

	payload, err := proto.Marshal(envelope)
	if err != nil {
		t.Fatalf("proto.Marshal() error = %v", err)
	}
	return encodedFrame(t, payload)
}
