package api

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Client struct {
	BaseURL string
	Token   string
}

func New(baseURL, token string) *Client {
	return &Client{BaseURL: baseURL, Token: token}
}

func (c *Client) Post(path string, body interface{}) (*http.Response, error) {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", c.BaseURL+path, bytes.NewBuffer(b))
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}
