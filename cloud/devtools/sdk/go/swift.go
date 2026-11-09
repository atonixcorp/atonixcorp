package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type Container struct {
	Name string `json:"name"`
}

type Object struct {
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
}

func (c *Client) CreateContainer(ctx context.Context, name string) (*Container, error) {
	body, status, err := c.do(ctx, "POST", "/swift/container/create", map[string]string{"name": name})
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, parseAPIError(status, body)
	}

	var cont Container
	if err := json.Unmarshal(body, &cont); err != nil {
		return nil, fmt.Errorf("decode container: %w", err)
	}
	return &cont, nil
}

func (c *Client) DeleteContainer(ctx context.Context, name string) error {
	body, status, err := c.do(ctx, "DELETE", "/swift/container/delete/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return parseAPIError(status, body)
	}
	return nil
}

func (c *Client) ListContainers(ctx context.Context) ([]Container, error) {
	body, status, err := c.do(ctx, "GET", "/swift/container/list", nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, parseAPIError(status, body)
	}
	var containers []Container
	if err := json.Unmarshal(body, &containers); err != nil {
		return nil, fmt.Errorf("decode containers: %w", err)
	}
	return containers, nil
}

func (c *Client) UploadObject(ctx context.Context, container, name string, data []byte) (*Object, error) {
	body, status, err := c.do(ctx, "POST", "/swift/object/upload", map[string]any{
		"container": container,
		"name":      name,
		"data":      data,
	})
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, parseAPIError(status, body)
	}

	var obj Object
	if err := json.Unmarshal(body, &obj); err != nil {
		return nil, fmt.Errorf("decode object: %w", err)
	}
	return &obj, nil
}

func (c *Client) DeleteObject(ctx context.Context, container, name string) error {
	body, status, err := c.do(ctx, "DELETE", "/swift/object/delete/"+url.PathEscape(container)+"/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return parseAPIError(status, body)
	}
	return nil
}
