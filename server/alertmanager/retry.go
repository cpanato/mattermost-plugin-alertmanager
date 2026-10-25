package alertmanager

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
)

func httpBackoff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 200 * time.Millisecond
	b.MaxInterval = 15 * time.Second
	b.MaxElapsedTime = 30 * time.Second
	return b
}

func httpRetry(method string, url string) (*http.Response, error) {
	var resp *http.Response
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)

	fn := func() error {
		req, errReq := http.NewRequest(method, url, nil)
		if errReq != nil {
			return backoff.Permanent(errReq)
		}

		resp, err = http.DefaultClient.Do(req.WithContext(ctx))
		if err != nil {
			// Retrying cannot fix certificate trust, expiry or hostname errors.
			// Return the original cause before the retry deadline obscures it.
			var certificateErr *tls.CertificateVerificationError
			if errors.As(err, &certificateErr) {
				return backoff.Permanent(err)
			}
			return err
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			resp.Body.Close()
			statusErr := fmt.Errorf("alertmanager returned HTTP %d", resp.StatusCode)
			if resp.StatusCode < 500 && resp.StatusCode != http.StatusTooManyRequests {
				return backoff.Permanent(statusErr)
			}
			return statusErr
		}

		return nil
	}

	if errRetry := backoff.Retry(fn, backoff.WithContext(httpBackoff(), ctx)); errRetry != nil {
		cancel()
		return nil, errRetry
	}

	// The caller still needs the context while decoding the response body.
	resp.Body = &responseBody{ReadCloser: resp.Body, cancel: cancel}
	return resp, err
}

type responseBody struct {
	io.ReadCloser
	cancel context.CancelFunc
}

func (b *responseBody) Close() error {
	defer b.cancel()
	return b.ReadCloser.Close()
}
