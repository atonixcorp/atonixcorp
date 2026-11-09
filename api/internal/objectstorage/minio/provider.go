package minio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"atonixcorp/api/internal/objectstorage"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Provider struct {
	client *minio.Client
	bucket string
}

// FromEnvironment returns nil when object storage is disabled.
func FromEnvironment() (*Provider, error) {
	provider := strings.TrimSpace(os.Getenv("STORAGE_PROVIDER"))
	if provider == "" {
		return nil, nil
	}
	if provider != "minio" {
		return nil, fmt.Errorf("unsupported object storage provider %q", provider)
	}
	endpoint := strings.TrimSpace(os.Getenv("MINIO_ENDPOINT"))
	accessKey := strings.TrimSpace(os.Getenv("MINIO_ACCESS_KEY"))
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucket := strings.TrimSpace(os.Getenv("MINIO_BUCKET"))
	if bucket == "" {
		bucket = "atonixcorp-bucket1"
	}
	if endpoint == "" || accessKey == "" || secretKey == "" {
		return nil, errors.New("MINIO_ENDPOINT, MINIO_ACCESS_KEY and MINIO_SECRET_KEY are required")
	}
	if strings.EqualFold(accessKey, "samorca") {
		return nil, errors.New("MinIO administrator credentials are not allowed")
	}
	secure := true
	if value := strings.TrimSpace(os.Getenv("MINIO_USE_SSL")); value != "" {
		var err error
		secure, err = strconv.ParseBool(value)
		if err != nil {
			return nil, fmt.Errorf("invalid MINIO_USE_SSL: %w", err)
		}
	}
	if !secure && os.Getenv("MINIO_ALLOW_INSECURE_DEVELOPMENT") != "true" {
		return nil, errors.New("unencrypted MinIO requires MINIO_ALLOW_INSECURE_DEVELOPMENT=true")
	}
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure, Region: strings.TrimSpace(os.Getenv("MINIO_REGION")),
	})
	if err != nil {
		return nil, fmt.Errorf("configure MinIO: %w", err)
	}
	return &Provider{client: client, bucket: bucket}, nil
}

func (p *Provider) Name() string   { return "minio" }
func (p *Provider) Bucket() string { return p.bucket }
func (p *Provider) Health(ctx context.Context) error {
	exists, err := p.client.BucketExists(ctx, p.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("object storage bucket does not exist")
	}
	return nil
}
func (p *Provider) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := p.client.PutObject(ctx, p.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}
func (p *Provider) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return p.get(ctx, key, "")
}
func (p *Provider) GetIfMatch(ctx context.Context, key, etag string) (io.ReadCloser, error) {
	if etag == "" {
		return nil, errors.New("missing object ETag")
	}
	return p.get(ctx, key, etag)
}
func (p *Provider) get(ctx context.Context, key, etag string) (io.ReadCloser, error) {
	opts := minio.GetObjectOptions{}
	if etag != "" {
		if err := opts.SetMatchETag(etag); err != nil {
			return nil, err
		}
	}
	obj, err := p.client.GetObject(ctx, p.bucket, key, opts)
	if err != nil {
		return nil, translate(err)
	}
	if _, err := obj.Stat(); err != nil {
		obj.Close()
		return nil, translate(err)
	}
	return obj, nil
}
func (p *Provider) Delete(ctx context.Context, key string) error {
	return p.client.RemoveObject(ctx, p.bucket, key, minio.RemoveObjectOptions{})
}
func (p *Provider) Exists(ctx context.Context, key string) (bool, error) {
	_, err := p.Stat(ctx, key)
	if errors.Is(err, objectstorage.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}
func (p *Provider) Stat(ctx context.Context, key string) (objectstorage.Info, error) {
	info, err := p.client.StatObject(ctx, p.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return objectstorage.Info{}, translate(err)
	}
	return objectstorage.Info{Key: info.Key, Size: info.Size, ContentType: info.ContentType, ETag: info.ETag}, nil
}
func (p *Provider) List(ctx context.Context, prefix string) ([]objectstorage.Info, error) {
	var result []objectstorage.Info
	for info := range p.client.ListObjects(ctx, p.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if info.Err != nil {
			return nil, info.Err
		}
		result = append(result, objectstorage.Info{Key: info.Key, Size: info.Size, ContentType: info.ContentType, ETag: info.ETag})
	}
	return result, nil
}
func (p *Provider) Copy(ctx context.Context, source, destination string) error {
	_, err := p.client.CopyObject(ctx, minio.CopyDestOptions{Bucket: p.bucket, Object: destination}, minio.CopySrcOptions{Bucket: p.bucket, Object: source})
	return translate(err)
}
func (p *Provider) CopyIfMatch(ctx context.Context, source, destination, etag string) error {
	if etag == "" {
		return errors.New("missing object ETag")
	}
	_, err := p.client.CopyObject(ctx, minio.CopyDestOptions{Bucket: p.bucket, Object: destination}, minio.CopySrcOptions{Bucket: p.bucket, Object: source, MatchETag: etag})
	return translate(err)
}
func (p *Provider) PresignGet(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
	return p.client.PresignedGetObject(ctx, p.bucket, key, expiry, url.Values{"response-content-disposition": {"attachment"}})
}
func (p *Provider) PresignPut(ctx context.Context, key string, expiry time.Duration) (*url.URL, error) {
	return p.client.PresignedPutObject(ctx, p.bucket, key, expiry)
}
func (p *Provider) PresignPost(ctx context.Context, key, contentType string, size int64, expires time.Time) (objectstorage.UploadForm, error) {
	policy := minio.NewPostPolicy()
	for _, err := range []error{
		policy.SetBucket(p.bucket), policy.SetKey(key), policy.SetExpires(expires),
		policy.SetContentType(contentType), policy.SetContentLengthRange(size, size),
	} {
		if err != nil {
			return objectstorage.UploadForm{}, err
		}
	}
	u, fields, err := p.client.PresignedPostPolicy(ctx, policy)
	if err != nil {
		return objectstorage.UploadForm{}, err
	}
	return objectstorage.UploadForm{URL: u.String(), Fields: fields}, nil
}
func translate(err error) error {
	if err != nil {
		code := minio.ToErrorResponse(err).Code
		if code == "NoSuchKey" || code == "NoSuchObject" || code == "NotFound" {
			return objectstorage.ErrNotFound
		}
	}
	return err
}
