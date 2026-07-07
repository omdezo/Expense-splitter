package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"expense-splitter/types"
)

var ErrNotConfigured = errors.New("storage: not configured")

// Client wraps the S3-compatible object store that holds proof images. The
// database keeps only metadata (sha256, size, key); bytes live here.
type Client struct {
	cfg types.StorageConfig

	mu     sync.Mutex
	minio  *minio.Client
	bucket bool
}

func New(cfg types.StorageConfig) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) client(ctx context.Context) (*minio.Client, error) {
	if !c.cfg.Enabled() {
		return nil, ErrNotConfigured
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.minio == nil {
		m, err := minio.New(c.cfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(c.cfg.AccessKey, c.cfg.SecretKey, ""),
			Secure: c.cfg.UseSSL,
		})
		if err != nil {
			return nil, err
		}
		c.minio = m
	}
	if !c.bucket {
		exists, err := c.minio.BucketExists(ctx, c.cfg.Bucket)
		if err != nil {
			return nil, err
		}
		if !exists {
			if err := c.minio.MakeBucket(ctx, c.cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
				return nil, err
			}
		}
		c.bucket = true
	}
	return c.minio, nil
}

func (c *Client) Put(ctx context.Context, key string, data []byte, contentType string) error {
	m, err := c.client(ctx)
	if err != nil {
		return err
	}
	_, err = m.PutObject(ctx, c.cfg.Bucket, key, bytes.NewReader(data), int64(len(data)),
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (c *Client) Get(ctx context.Context, key string) ([]byte, error) {
	m, err := c.client(ctx)
	if err != nil {
		return nil, err
	}
	obj, err := m.GetObject(ctx, c.cfg.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(obj)
}
