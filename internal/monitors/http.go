package monitors

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/monocle-dev/monocle/internal/types"
)

func GetHTTP(config *types.HttpConfig) error {
	client := &http.Client{
		Timeout: time.Duration(config.Timeout) * time.Second,
	}

	req, err := http.NewRequest(config.Method, config.URL, nil)

	if err != nil {
		return err
	}

	for key, value := range config.Headers {
		req.Header.Add(key, value)
	}

	req = req.WithContext(context.Background())

	resp, err := client.Do(req)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != config.ExpectedStatus {
		return errors.New("unexpected status code: " + resp.Status)
	}

	return nil
}
