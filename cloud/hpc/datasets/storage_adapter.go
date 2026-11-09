package datasets

import (
	"context"
	"fmt"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type StorageAdapter interface {
	Upload(ctx context.Context, path string, r io.Reader, size int64, contentType string) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
}

type S3Adapter struct {
	client     *minio.Client
	bucketName string
}

func NewS3Adapter(endpoint, accessKey, secretKey, bucket string, secure bool) (*S3Adapter, error) {
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}

	return &S3Adapter{
		client:     client,
		bucketName: bucket,
	}, nil
}

func (s *S3Adapter) Upload(ctx context.Context, path string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucketName, path, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	return err
}

func (s *S3Adapter) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return s.client.GetObject(ctx, s.bucketName, path, minio.GetObjectOptions{})
}

func (s *S3Adapter) EnsureBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.MakeBucket(ctx, s.bucketName, minio.MakeBucketOptions{})
}

func (s *S3Adapter) CheckBucket(ctx context.Context) error {
	exists, err := s.client.BucketExists(ctx, s.bucketName)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("dataset bucket %q does not exist", s.bucketName)
	}
	return nil
}

func (s *S3Adapter) Delete(ctx context.Context, path string) error {
	return s.client.RemoveObject(ctx, s.bucketName, path, minio.RemoveObjectOptions{})
}
