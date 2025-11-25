package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"
)

func (c *Client) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *Client) GetJSON(ctx context.Context, url string, dest any) (int, error) {
	resp, err := c.Get(ctx, url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}

	if len(body) == 0 {
		return resp.StatusCode, nil
	}

	if err := json.Unmarshal(body, dest); err != nil {
		return resp.StatusCode, err
	}

	return resp.StatusCode, nil
}

func (c *Client) Delete(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(req)
}

func (c *Client) PostBytes(ctx context.Context, url, contentType string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return c.Do(req)
}

func (c *Client) PostJSON(ctx context.Context, url string, v any) (*http.Response, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return c.PostBytes(ctx, url, "application/json", data)
}

func (c *Client) GetString(ctx context.Context, url string) (string, int, error) {
	b, status, err := c.GetBytes(ctx, url)
	return string(b), status, err
}

func (c *Client) PostForm(ctx context.Context, targetURL string, values map[string]string) (*http.Response, error) {
	if len(values) == 0 {
		return c.PostBytes(ctx, targetURL, "application/x-www-form-urlencoded", nil)
	}
	form := url.Values{}
	for k, v := range values {
		form.Set(k, v)
	}
	body := form.Encode()
	return c.PostBytes(ctx, targetURL, "application/x-www-form-urlencoded", []byte(body))
}

func (c *Client) GetBytesURL(targetURL string) ([]byte, int, error) {
	return c.GetBytes(context.Background(), targetURL)
}

func (c *Client) GetBytesTimeout(targetURL string, timeout time.Duration) ([]byte, int, error) {
	if timeout <= 0 {
		return nil, 0, context.DeadlineExceeded
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return c.GetBytes(ctx, targetURL)
}

func (c *Client) GetBytesDeadline(targetURL string, deadline time.Time) ([]byte, int, error) {
	ctx := context.Background()
	if !deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}
	return c.GetBytes(ctx, targetURL)
}

func (c *Client) DoTimeout(req *http.Request, timeout time.Duration) (*http.Response, error) {
	if timeout <= 0 {
		return nil, context.DeadlineExceeded
	}
	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req = req.WithContext(ctx)
	return c.Do(req)
}

func (c *Client) DoDeadline(req *http.Request, deadline time.Time) (*http.Response, error) {
	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if !deadline.IsZero() {
		var cancel context.CancelFunc
		ctx, cancel = context.WithDeadline(ctx, deadline)
		defer cancel()
	}

	req = req.WithContext(ctx)
	return c.Do(req)
}
