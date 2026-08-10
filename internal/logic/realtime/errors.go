package realtime

import "errors"

var (
	errInvalidFrameMagic       = errors.New("invalid frame magic")
	errUnsupportedFrameVersion = errors.New("unsupported frame version")
	errInvalidFrameLength      = errors.New("invalid frame length")
	errFrameChecksumMismatch   = errors.New("frame checksum mismatch")
	errInvalidRequestID        = errors.New("invalid request id")
	errInvalidPayload          = errors.New("invalid payload")
)
