package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type ComputeInstance struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Plan  string `json:"plan"`
	State string `json:"state"`
}

type CreateComputeRequest struct {
	Name string `json:"name"`
	Plan string `json:"plan"`
}

func (c *Client) CreateCompute(ctx context.Context, req CreateComputeRequest) (*ComputeInstance, error) {
	body, status, err := c.do(ctx, "POST", "/compute/create", req)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, parseAPIError(status, body)
	}

	var inst ComputeInstance
	if err := json.Unmarshal(body, &inst); err != nil {
		return nil, fmt.Errorf("decode compute instance: %w", err)
	}
	return &inst, nil
}

func (c *Client) ListCompute(ctx context.Context) ([]ComputeInstance, error) {
	body, status, err := c.do(ctx, "GET", "/compute/list", nil)
	if err != nil {
		return nil, err
	}
	if status >= 400 {
		return nil, parseAPIError(status, body)
	}

	var list []ComputeInstance
	if err := json.Unmarshal(body, &list); err != nil {
		return nil, fmt.Errorf("decode compute list: %w", err)
	}
	return list, nil
}

func (c *Client) DeleteCompute(ctx context.Context, id string) error {
	body, status, err := c.do(ctx, "DELETE", "/compute/delete/"+url.PathEscape(id), nil)
	if err != nil {
		return err
	}
	if status >= 400 {
		return parseAPIError(status, body)
	}
	return nil
}
