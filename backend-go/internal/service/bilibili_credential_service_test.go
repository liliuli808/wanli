package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wanli/backend-go/internal/infrastructure/bilibili"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/pkg/secretbox"
)

type fakeCredentialStore struct {
	cred    *model.BilibiliCredential
	setErr  error
	getErr  error
	cleared bool
}

func (f *fakeCredentialStore) GetBilibiliCredential(context.Context, uuid.UUID) (*model.BilibiliCredential, error) {
	return f.cred, f.getErr
}

func (f *fakeCredentialStore) SetBilibiliCredential(_ context.Context, _ uuid.UUID, enc, mid, uname string) error {
	if f.setErr != nil {
		return f.setErr
	}
	now := time.Now()
	f.cred = &model.BilibiliCredential{EncSessdata: enc, Mid: mid, Uname: uname, UpdatedAt: &now}
	return nil
}

func (f *fakeCredentialStore) ClearBilibiliCredential(context.Context, uuid.UUID) error {
	f.cleared = true
	f.cred = nil
	return nil
}

func newNavServer(t *testing.T, loggedIn bool) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/x/web-interface/nav", r.URL.Path)
		if loggedIn {
			fmt.Fprint(w, `{"code":0,"data":{"isLogin":true,"mid":123456,"uname":"测试用户"}}`)
			return
		}
		fmt.Fprint(w, `{"code":-101,"message":"账号未登录","data":{"isLogin":false}}`)
	}))
	t.Cleanup(server.Close)
	return server
}

func newCredentialService(t *testing.T, store BilibiliCredentialStore, nav *httptest.Server, withBox bool) *BilibiliCredentialService {
	t.Helper()
	var box *secretbox.Box
	if withBox {
		var err error
		box, err = secretbox.New("test-key")
		require.NoError(t, err)
	}
	return NewBilibiliCredentialService(store, box, bilibili.New(nav.URL, nil), nil)
}

func TestBilibiliCredential_BindValid(t *testing.T) {
	store := &fakeCredentialStore{}
	svc := newCredentialService(t, store, newNavServer(t, true), true)
	userID := uuid.New()

	status, err := svc.Bind(context.Background(), userID, "abc123%2C1793262567%2Cxyz*11")
	require.NoError(t, err)
	assert.True(t, status.Bound)
	assert.Equal(t, "123456", status.Mid)
	assert.Equal(t, "测试用户", status.Uname)

	// 落库的必须是密文，且能解出原始 SESSDATA
	require.NotNil(t, store.cred)
	assert.NotContains(t, store.cred.EncSessdata, "abc123")
	assert.Equal(t, "abc123%2C1793262567%2Cxyz*11", svc.Sessdata(context.Background(), userID))
}

func TestBilibiliCredential_BindExpired(t *testing.T) {
	svc := newCredentialService(t, &fakeCredentialStore{}, newNavServer(t, false), true)

	_, err := svc.Bind(context.Background(), uuid.New(), "abc123%2C1793262567%2Cxyz*11")
	assert.ErrorIs(t, err, ErrBilibiliCredentialInvalid)
}

func TestBilibiliCredential_BindRejectsMalformedInput(t *testing.T) {
	svc := newCredentialService(t, &fakeCredentialStore{}, newNavServer(t, true), true)

	for _, input := range []string{
		"",
		"short",
		"has space in it 12345",
		"abc;other=1;",
		"abc\n1234567890",
	} {
		_, err := svc.Bind(context.Background(), uuid.New(), input)
		assert.ErrorIs(t, err, ErrBilibiliCredentialInvalid, "输入: %q", input)
	}
}

func TestBilibiliCredential_BindNormalizesPrefix(t *testing.T) {
	store := &fakeCredentialStore{}
	svc := newCredentialService(t, store, newNavServer(t, true), true)

	_, err := svc.Bind(context.Background(), uuid.New(), `  SESSDATA="abc123%2C1793262567%2Cxyz*11"  `)
	require.NoError(t, err)

	sessdata, err := secretbox.New("test-key")
	require.NoError(t, err)
	assert.Equal(t, "abc123%2C1793262567%2Cxyz*11", func() string {
		plain, err := sessdata.Decrypt(store.cred.EncSessdata)
		require.NoError(t, err)
		return plain
	}())
}

func TestBilibiliCredential_DisabledWithoutKey(t *testing.T) {
	svc := newCredentialService(t, &fakeCredentialStore{}, newNavServer(t, true), false)

	_, err := svc.Bind(context.Background(), uuid.New(), "abc123%2C1793262567%2Cxyz*11")
	assert.ErrorIs(t, err, ErrBilibiliCredentialDisabled)
	assert.ErrorIs(t, svc.Unbind(context.Background(), uuid.New()), ErrBilibiliCredentialDisabled)
	assert.Equal(t, "", svc.Sessdata(context.Background(), uuid.New()))
}

func TestBilibiliCredential_StatusAndUnbind(t *testing.T) {
	store := &fakeCredentialStore{}
	svc := newCredentialService(t, store, newNavServer(t, true), true)
	userID := uuid.New()

	status, err := svc.Status(context.Background(), userID)
	require.NoError(t, err)
	assert.False(t, status.Bound)

	_, err = svc.Bind(context.Background(), userID, "abc123%2C1793262567%2Cxyz*11")
	require.NoError(t, err)

	require.NoError(t, svc.Unbind(context.Background(), userID))
	assert.True(t, store.cleared)

	status, err = svc.Status(context.Background(), userID)
	require.NoError(t, err)
	assert.False(t, status.Bound)
}

func TestBilibiliCredential_SessdataOnDecryptFailure(t *testing.T) {
	store := &fakeCredentialStore{cred: &model.BilibiliCredential{EncSessdata: "not-a-valid-token"}}
	svc := newCredentialService(t, store, newNavServer(t, true), true)

	assert.Equal(t, "", svc.Sessdata(context.Background(), uuid.New()), "解密失败应降级为匿名")
}
