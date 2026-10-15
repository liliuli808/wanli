package service

import (
	"context"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/config"
)

type fakePutObjectPresigner struct {
	bucket string
	object string
	expiry time.Duration
}

func (f *fakePutObjectPresigner) PresignedPutObject(
	_ context.Context,
	bucket, object string,
	expiry time.Duration,
) (*url.URL, error) {
	f.bucket = bucket
	f.object = object
	f.expiry = expiry
	return url.Parse("http://10.0.2.2:9000/signed-upload")
}

func TestUploadService_PresignsRestrictedCookingPhoto(t *testing.T) {
	presigner := &fakePutObjectPresigner{}
	svc := NewUploadService(presigner, config.StorageConfig{
		Endpoint:       "127.0.0.1:9000",
		PublicEndpoint: "10.0.2.2:9000",
		Bucket:         "wanli-assets",
	})

	result, err := svc.PresignCookingPhoto(context.Background(), "image/jpeg", 1024)
	require.NoError(t, err)
	assert.Equal(t, "wanli-assets", presigner.bucket)
	assert.Contains(t, presigner.object, "uploads/records/")
	assert.Contains(t, presigner.object, ".jpg")
	assert.Equal(t, 15*time.Minute, presigner.expiry)
	assert.Equal(t, "http://10.0.2.2:9000/wanli-assets/"+presigner.object, result.PublicURL)
	assert.Equal(t, 900, result.ExpiresIn)
}

func TestUploadService_RejectsInvalidTypeAndSize(t *testing.T) {
	svc := NewUploadService(&fakePutObjectPresigner{}, config.StorageConfig{Bucket: "wanli-assets"})

	_, err := svc.PresignCookingPhoto(context.Background(), "application/pdf", 100)
	assert.ErrorIs(t, err, ErrUploadInvalid)

	_, err = svc.PresignCookingPhoto(context.Background(), "image/png", MaxCookingPhotoBytes+1)
	assert.ErrorIs(t, err, ErrUploadInvalid)
}
