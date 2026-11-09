package objectstorage

import (
	"context"
	"errors"
	"io"
	"net/url"
	"time"
)

var ErrNotFound = errors.New("object not found")

type Info struct {
	Key         string
	Size        int64
	ContentType string
	ETag        string
}

type UploadForm struct {
	URL    string            `json:"url"`
	Fields map[string]string `json:"fields"`
}

// Provider operates only on the configured bucket; callers enforce authorization.
type Provider interface {
	Name() string
	Bucket() string
	Health(context.Context) error
	Put(context.Context, string, io.Reader, int64, string) error
	Get(context.Context, string) (io.ReadCloser, error)
	GetIfMatch(context.Context, string, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
	Exists(context.Context, string) (bool, error)
	Stat(context.Context, string) (Info, error)
	List(context.Context, string) ([]Info, error)
	Copy(context.Context, string, string) error
	CopyIfMatch(context.Context, string, string, string) error
	PresignGet(context.Context, string, time.Duration) (*url.URL, error)
	PresignPut(context.Context, string, time.Duration) (*url.URL, error)
	PresignPost(context.Context, string, string, int64, time.Time) (UploadForm, error)
}
