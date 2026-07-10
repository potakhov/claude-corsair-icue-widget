package client

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"xeneoncc/internal/protocol"
)

type Client struct {
	BaseURL string
	Token   string
}

func New(baseURL, token string) *Client {
	return &Client{BaseURL: baseURL, Token: token}
}

func (c *Client) post(path string, body any, timeout time.Duration) (*http.Response, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Bridge-Token", c.Token)
	req.Header.Set("Content-Type", "application/json")
	return (&http.Client{Timeout: timeout}).Do(req)
}

func (c *Client) PostUsage(u protocol.Usage) error {
	resp, err := c.post("/v1/usage", u, 3*time.Second)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func (c *Client) PostNotify(n protocol.Notification) error {
	resp, err := c.post("/v1/notify", n, 3*time.Second)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
