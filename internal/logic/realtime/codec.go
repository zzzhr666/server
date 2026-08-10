package realtime

import (
	"io"
	"server/internal/contract/realtimepb"

	"google.golang.org/protobuf/proto"
)

func readClientEnvelope(reader io.Reader) (*realtimepb.ClientEnvelope, error) {
	data, err := readFrame(reader)
	if err != nil {
		return nil, err
	}
	var clientEnvelope realtimepb.ClientEnvelope
	err = proto.Unmarshal(data, &clientEnvelope)
	if err != nil {
		return nil, err
	}

	if clientEnvelope.GetRequestId() == 0 {
		return nil, errInvalidRequestID
	}
	if clientEnvelope.GetPayload() == nil {
		return nil, errInvalidPayload
	}
	return &clientEnvelope, nil
}

func writeServerEnvelope(writer io.Writer, serverEnvelope *realtimepb.ServerEnvelope) error {
	if serverEnvelope.GetPayload() == nil {
		return errInvalidPayload
	}
	data, err := proto.Marshal(serverEnvelope)
	if err != nil {
		return err
	}
	return writeFrame(writer, data)
}
