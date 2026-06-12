package data

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var ErrS3UploadConfigInvalid = errors.New("s3 upload config invalid")

type S3UploadConfig struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	PublicBaseURL   string
	PresignExpire   time.Duration
	UsePathStyle    bool
	Prefix          string
}

type UploadObject struct {
	FileName    string
	ContentType string
	Size        int64
	Body        io.Reader
}

type UploadedObject struct {
	Key string
	URL string
}

type S3UploadData struct {
	config        S3UploadConfig
	client        *s3.Client
	presignClient *s3.PresignClient
}

func NewS3UploadData(config S3UploadConfig) *S3UploadData {
	if config.Region == "" {
		config.Region = "us-east-1"
	}
	if config.PresignExpire <= 0 {
		config.PresignExpire = time.Hour
	}

	awsConfig := aws.Config{
		Region: config.Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.SecretAccessKey,
			"",
		)),
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		if config.Endpoint != "" {
			options.BaseEndpoint = aws.String(config.Endpoint)
		}
		options.UsePathStyle = config.UsePathStyle
	})

	return &S3UploadData{
		config:        config,
		client:        client,
		presignClient: s3.NewPresignClient(client),
	}
}

// Upload 将文件流写入 S3 兼容对象存储，并返回前端可直接访问的对象地址。
func (s *S3UploadData) Upload(ctx context.Context, object UploadObject) (*UploadedObject, error) {
	if s.config.Bucket == "" || s.config.AccessKeyID == "" || s.config.SecretAccessKey == "" {
		return nil, ErrS3UploadConfigInvalid
	}

	key, err := buildObjectKey(s.config.Prefix, object.FileName)
	if err != nil {
		return nil, err
	}

	contentType := object.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(object.FileName))
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(s.config.Bucket),
		Key:           aws.String(key),
		Body:          object.Body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(object.Size),
	})
	if err != nil {
		return nil, err
	}

	objectURL, err := s.buildObjectURL(ctx, key)
	if err != nil {
		return nil, err
	}

	return &UploadedObject{
		Key: key,
		URL: objectURL,
	}, nil
}

func buildObjectKey(prefix, fileName string) (string, error) {
	random, err := randomHex(8)
	if err != nil {
		return "", err
	}

	baseName := filepath.Base(strings.TrimSpace(fileName))
	if baseName == "" || baseName == "." {
		baseName = "file"
	}
	baseName = strings.NewReplacer("/", "-", "\\", "-", ":", "-").Replace(baseName)

	datePath := time.Now().Format("2006/01/02")
	objectName := fmt.Sprintf("%s-%s", random, baseName)
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return path.Join(datePath, objectName), nil
	}

	return path.Join(prefix, datePath, objectName), nil
}

func randomHex(size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}

func (s *S3UploadData) buildObjectURL(ctx context.Context, key string) (string, error) {
	// 存储桶无法开放公共读权限时，使用 access key/secret 生成临时 GET 链接给前端访问。
	request, err := s.presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.config.Bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(s.config.PresignExpire))
	if err != nil {
		return "", err
	}

	return request.URL, nil
}
