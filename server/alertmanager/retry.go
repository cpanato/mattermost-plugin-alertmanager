package alertmanager

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
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
	defer cancel()

	fn := func() error {
		req, err := http.NewRequest(method, url, nil)
		if err != nil {
			return err
		}

		req = req.WithContext(ctx)

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		switch method {
		case http.MethodGet:
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("status code is %d not 200", resp.StatusCode)
			}
		case http.MethodPost:
			if resp.StatusCode == http.StatusBadRequest {
				return fmt.Errorf("status code is %d not 3xx", resp.StatusCode)
			}
		}

		return nil
	}

	if err := backoff.Retry(fn, httpBackoff()); err != nil {
		return nil, err
	}
	return resp, err
}
