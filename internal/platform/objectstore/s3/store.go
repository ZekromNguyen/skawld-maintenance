package s3

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/config"
	"github.com/ZekromNguyen/skawld-maintenance/internal/platform/objectstore"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type Store struct {
	bucket  string
	client  *awss3.Client
	presign *awss3.PresignClient
}

func New(ctx context.Context, cfg config.ObjectStore) (*Store, error) {
	options := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" {
		options = append(options, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID, cfg.SecretAccessKey, "",
			),
		))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("load S3 configuration: %w", err)
	}
	client := awss3.NewFromConfig(awsCfg, func(options *awss3.Options) {
		options.UsePathStyle = cfg.UsePathStyle
		if cfg.Endpoint != "" {
			options.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})
	return &Store{
		bucket:  cfg.Bucket,
		client:  client,
		presign: awss3.NewPresignClient(client),
	}, nil
}

func (s *Store) Put(
	ctx context.Context,
	request objectstore.PutRequest,
) (objectstore.Object, error) {
	result, err := s.client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(request.Key),
		Body: request.Body, ContentLength: aws.Int64(request.Size),
		ContentType: aws.String(request.ContentType), Metadata: request.Metadata,
	})
	if err != nil {
		return objectstore.Object{}, err
	}
	return objectstore.Object{
		Key: request.Key, Size: request.Size, ContentType: request.ContentType,
		ETag: aws.ToString(result.ETag), Metadata: request.Metadata,
	}, nil
}

func (s *Store) Get(
	ctx context.Context,
	key string,
) (io.ReadCloser, objectstore.Object, error) {
	result, err := s.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	if err != nil {
		return nil, objectstore.Object{}, err
	}
	return result.Body, objectstore.Object{
		Key: key, Size: aws.ToInt64(result.ContentLength),
		ContentType: aws.ToString(result.ContentType),
		ETag:        aws.ToString(result.ETag), Metadata: result.Metadata,
	}, nil
}

func (s *Store) Head(ctx context.Context, key string) (objectstore.Object, error) {
	result, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	if err != nil {
		return objectstore.Object{}, err
	}
	return objectstore.Object{
		Key: key, Size: aws.ToInt64(result.ContentLength),
		ContentType: aws.ToString(result.ContentType),
		ETag:        aws.ToString(result.ETag), Metadata: result.Metadata,
	}, nil
}

func (s *Store) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	})
	return err
}

func (s *Store) SignedUploadURL(
	ctx context.Context,
	key string,
	constraints objectstore.UploadConstraints,
	expiresIn time.Duration,
) (objectstore.SignedURL, error) {
	result, err := s.presign.PresignPutObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
		ContentType:   aws.String(constraints.ContentType),
		ContentLength: aws.Int64(constraints.Size),
		Metadata:      constraints.Metadata,
	}, awss3.WithPresignExpires(expiresIn))
	if err != nil {
		return objectstore.SignedURL{}, err
	}
	headers := make(map[string]string, len(result.SignedHeader))
	for name, values := range result.SignedHeader {
		if strings.EqualFold(name, "Host") {
			continue
		}
		if len(values) > 0 {
			headers[name] = values[0]
		}
	}
	return objectstore.SignedURL{
		URL: result.URL, Headers: headers, ExpiresAt: time.Now().UTC().Add(expiresIn),
	}, nil
}

func (s *Store) SignedDownloadURL(
	ctx context.Context,
	key string,
	expiresIn time.Duration,
) (string, error) {
	result, err := s.presign.PresignGetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
	}, awss3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

var _ objectstore.Store = (*Store)(nil)
