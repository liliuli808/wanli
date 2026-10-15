package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/config"
)

const MaxCookingPhotoBytes int64 = 10 * 1024 * 1024

var ErrUploadInvalid = errors.New("上传参数校验失败")

type PutObjectPresigner interface {
	PresignedPutObject(
		ctx context.Context,
		bucketName, objectName string,
		expiry time.Duration,
	) (*url.URL, error)
}

type PresignedUpload struct {
	UploadURL  string
	PublicURL  string
	ObjectName string
	ExpiresIn  int
}

type UploadService struct {
	presigner PutObjectPresigner
	cfg       config.StorageConfig
}

func NewUploadService(presigner PutObjectPresigner, cfg config.StorageConfig) *UploadService {
	return &UploadService{presigner: presigner, cfg: cfg}
}

func (s *UploadService) PresignCookingPhoto(
	ctx context.Context,
	contentType string,
	sizeBytes int64,
) (*PresignedUpload, error) {
	extension, ok := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
	}[contentType]
	if !ok {
		return nil, fmt.Errorf("%w: 仅支持 JPEG、PNG、WebP", ErrUploadInvalid)
	}
	if sizeBytes < 1 || sizeBytes > MaxCookingPhotoBytes {
		return nil, fmt.Errorf("%w: 图片大小须在 1 字节到 10MB 之间", ErrUploadInvalid)
	}

	objectName := "uploads/records/" + uuid.NewString() + extension
	const expires = 15 * time.Minute
	signedURL, err := s.presigner.PresignedPutObject(ctx, s.cfg.Bucket, objectName, expires)
	if err != nil {
		return nil, fmt.Errorf("presign cooking photo: %w", err)
	}

	publicURL := CookingPhotoPublicPrefix(s.cfg) + path.Base(objectName)

	return &PresignedUpload{
		UploadURL:  signedURL.String(),
		PublicURL:  publicURL,
		ObjectName: objectName,
		ExpiresIn:  int(expires.Seconds()),
	}, nil
}

func CookingPhotoPublicPrefix(cfg config.StorageConfig) string {
	publicEndpoint := cfg.PublicEndpoint
	useSSL := cfg.UseSSL
	if publicEndpoint != "" {
		// 配置了对外地址时，协议由 PublicUseSSL 决定（内网 http / 公网 https 可并存）
		useSSL = cfg.PublicUseSSL
	} else {
		publicEndpoint = cfg.Endpoint
	}
	scheme := "http"
	if useSSL {
		scheme = "https"
	}
	return (&url.URL{
		Scheme: scheme,
		Host:   publicEndpoint,
		Path:   path.Join("/", cfg.Bucket, "uploads/records") + "/",
	}).String()
}
