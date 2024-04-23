package network

import (
	"context"
	"fmt"
	"github.com/avast/retry-go/v4"
	"net"
	"time"
)

func WaitForLocalPort(ctx context.Context, port int) {
	dialer := net.Dialer{
		Timeout: 10 * time.Second,
	}

	var conn net.Conn
	var err error

	_ = retry.Do(
		func() error {
			conn, err = dialer.DialContext(ctx, "tcp", fmt.Sprintf("localhost:%d", port))
			if err != nil {
				return err
			}

			_ = conn.Close()

			return nil
		},
		retry.Delay(1*time.Second), retry.MaxDelay(1*time.Second),
		retry.Attempts(0), retry.LastErrorOnly(true),
		retry.Context(ctx),
	)
}
