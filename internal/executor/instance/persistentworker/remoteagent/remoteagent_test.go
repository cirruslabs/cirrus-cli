package remoteagent

import (
	"context"
	"net"
	"testing"

	"github.com/cirruslabs/cirrus-cli/internal/logger"
)

type closeTrackingListener struct {
	closed bool
}

func (listener *closeTrackingListener) Accept() (net.Conn, error) {
	return nil, net.ErrClosed
}

func (listener *closeTrackingListener) Close() error {
	listener.closed = true

	return nil
}

func (listener *closeTrackingListener) Addr() net.Addr {
	return testAddr("test")
}

type testAddr string

func (addr testAddr) Network() string {
	return string(addr)
}

func (addr testAddr) String() string {
	return string(addr)
}

func TestCloseForwardingListenerSkipsAfterContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	listener := &closeTrackingListener{}
	closeForwardingListener(ctx, listener, &logger.LightweightStub{})

	if listener.closed {
		t.Fatal("expected listener close to be skipped after context cancellation")
	}
}

func TestCloseForwardingListenerClosesActiveContext(t *testing.T) {
	listener := &closeTrackingListener{}
	closeForwardingListener(context.Background(), listener, &logger.LightweightStub{})

	if !listener.closed {
		t.Fatal("expected listener to be closed while context is active")
	}
}
