package libp2pquic

import (
	"errors"
	"math"

	"github.com/libp2p/go-libp2p/core/network"

	"github.com/quic-go/quic-go"
)

const (
	reset quic.StreamErrorCode = 0
)

type stream struct {
	quic.Stream
}

var _ network.MuxedStream = &stream{}

func parseStreamError(err error) error {
	se := &quic.StreamError{}
	if err != nil && errors.As(err, &se) {
		code := se.ErrorCode
		if code > math.MaxUint32 {
			code = 0
		}
		err = &network.StreamError{
			ErrorCode: network.StreamErrorCode(code),
			Remote:    se.Remote,
		}
	}
	return err
}

func (s *stream) Read(b []byte) (n int, err error) {
	n, err = s.Stream.Read(b)
	return n, parseStreamError(err)
}

func (s *stream) Write(b []byte) (n int, err error) {
	n, err = s.Stream.Write(b)
	return n, parseStreamError(err)
}

func (s *stream) Reset() error {
	s.Stream.CancelRead(reset)
	s.Stream.CancelWrite(reset)
	return nil
}

func (s *stream) ResetWithError(errCode network.StreamErrorCode) error {
	s.Stream.CancelRead(quic.StreamErrorCode(errCode))
	s.Stream.CancelWrite(quic.StreamErrorCode(errCode))
	return nil
}

func (s *stream) Close() error {
	s.Stream.CancelRead(reset)
	return s.Stream.Close()
}

func (s *stream) CloseRead() error {
	s.Stream.CancelRead(reset)
	return nil
}

func (s *stream) CloseWrite() error {
	return s.Stream.Close()
}
