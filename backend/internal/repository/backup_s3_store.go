package repository

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// S3BackupStore implements service.BackupObjectStore using AWS S3 compatible storage
type S3BackupStore struct {
	client *s3.Client
	bucket string
}

// NewS3BackupStoreFactory returns a BackupObjectStoreFactory that creates S3-backed stores
func NewS3BackupStoreFactory() service.BackupObjectStoreFactory {
	return func(ctx context.Context, cfg *service.BackupS3Config) (service.BackupObjectStore, error) {
		region := cfg.Region
		if region == "" {
			region = "auto" // Cloudflare R2 默认 region
		}

		awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("load aws config: %w", err)
		}

		client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			if cfg.Endpoint != "" {
				o.BaseEndpoint = &cfg.Endpoint
			}
			if cfg.ForcePathStyle {
				o.UsePathStyle = true
			}
			o.APIOptions = append(o.APIOptions, v4.SwapComputePayloadSHA256ForUnsignedPayloadMiddleware)
			o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		})

		return &S3BackupStore{client: client, bucket: cfg.Bucket}, nil
	}
}

func (s *S3BackupStore) Upload(ctx context.Context, key string, body io.Reader, contentType string) (int64, error) {
	// 读取全部内容以获取大小（S3 PutObject 需要知道内容长度）
	// 注意：阿里云 OSS 不兼容 s3manager 分片上传的签名方式，因此使用 PutObject
	data, err := io.ReadAll(body)
	if err != nil {
		return 0, fmt.Errorf("read body: %w", err)
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.bucket,
		Key:         &key,
		Body:        bytes.NewReader(data),
		ContentType: &contentType,
	})
	if err != nil {
		return 0, fmt.Errorf("S3 PutObject: %w", err)
	}
	return int64(len(data)), nil
}

func (s *S3BackupStore) UploadMultipart(
	ctx context.Context,
	key string,
	body io.Reader,
	contentType string,
	partSize int64,
	progress func(service.BackupMultipartProgress) error,
) (service.BackupUploadResult, error) {
	if partSize < 5<<20 {
		partSize = 8 << 20
	}
	createResult, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:      &s.bucket,
		Key:         &key,
		ContentType: &contentType,
	})
	if err != nil {
		return service.BackupUploadResult{}, fmt.Errorf("S3 CreateMultipartUpload: %w", err)
	}
	uploadID := aws.ToString(createResult.UploadId)
	if uploadID == "" {
		return service.BackupUploadResult{}, fmt.Errorf("S3 CreateMultipartUpload returned empty upload id")
	}

	digest := sha256.New()
	parts := make([]types.CompletedPart, 0, 8)
	var uploadedBytes int64
	partNumber := int32(1)
	buf := make([]byte, partSize)
	abort := func(cause error) (service.BackupUploadResult, error) {
		_ = s.AbortMultipart(context.Background(), key, uploadID)
		return service.BackupUploadResult{UploadID: uploadID}, cause
	}

	for {
		n, readErr := io.ReadFull(body, buf)
		if n > 0 {
			part := append([]byte(nil), buf[:n]...)
			if _, err := digest.Write(part); err != nil {
				return abort(fmt.Errorf("hash multipart part: %w", err))
			}
			result, uploadErr := s.client.UploadPart(ctx, &s3.UploadPartInput{
				Bucket:     &s.bucket,
				Key:        &key,
				UploadId:   &uploadID,
				PartNumber: aws.Int32(partNumber),
				Body:       bytes.NewReader(part),
			})
			if uploadErr != nil {
				return abort(fmt.Errorf("S3 UploadPart %d: %w", partNumber, uploadErr))
			}
			etag := aws.ToString(result.ETag)
			parts = append(parts, types.CompletedPart{ETag: &etag, PartNumber: aws.Int32(partNumber)})
			uploadedBytes += int64(n)
			if progress != nil {
				if progressErr := progress(service.BackupMultipartProgress{
					UploadID:      uploadID,
					PartNumber:    int(partNumber),
					PartSizeBytes: int64(n),
					UploadedBytes: uploadedBytes,
					PartETag:      etag,
				}); progressErr != nil {
					return abort(progressErr)
				}
			}
			partNumber++
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return abort(fmt.Errorf("read multipart body: %w", readErr))
		}
	}

	if len(parts) == 0 {
		return abort(fmt.Errorf("multipart body is empty"))
	}
	_, err = s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   &s.bucket,
		Key:      &key,
		UploadId: &uploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: parts,
		},
	})
	if err != nil {
		return abort(fmt.Errorf("S3 CompleteMultipartUpload: %w", err))
	}

	return service.BackupUploadResult{
		UploadID:  uploadID,
		SizeBytes: uploadedBytes,
		SHA256:    hex.EncodeToString(digest.Sum(nil)),
		PartCount: len(parts),
	}, nil
}

func (s *S3BackupStore) AbortMultipart(ctx context.Context, key, uploadID string) error {
	if strings.TrimSpace(uploadID) == "" {
		return nil
	}
	_, err := s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   &s.bucket,
		Key:      &key,
		UploadId: &uploadID,
	})
	return err
}

func (s *S3BackupStore) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	if err != nil {
		return nil, fmt.Errorf("S3 GetObject: %w", err)
	}
	return result.Body, nil
}

func (s *S3BackupStore) Delete(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	return err
}

func (s *S3BackupStore) PresignURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)
	result, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", fmt.Errorf("presign url: %w", err)
	}
	return result.URL, nil
}

func (s *S3BackupStore) HeadBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &s.bucket,
	})
	if err != nil {
		return fmt.Errorf("S3 HeadBucket failed: %w", err)
	}
	return nil
}
