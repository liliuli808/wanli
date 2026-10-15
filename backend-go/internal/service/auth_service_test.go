package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/wanli/backend-go/internal/auth"
	"github.com/wanli/backend-go/internal/infrastructure/turnstile"
	"github.com/wanli/backend-go/internal/model"
)

// inMemoryUserRepo 是测试用的内存用户仓库。
type inMemoryUserRepo struct {
	users              map[string]*model.User
	updateProfileCalls int
	updatedNickname    string
	updatedAvatar      *string
}

func newInMemoryUserRepo() *inMemoryUserRepo {
	return &inMemoryUserRepo{users: make(map[string]*model.User)}
}

func (r *inMemoryUserRepo) Create(ctx context.Context, user *model.User) error {
	r.users[user.Email] = user
	return nil
}

func (r *inMemoryUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	return r.users[email], nil
}

func (r *inMemoryUserRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, nil
}

func (r *inMemoryUserRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	_, ok := r.users[email]
	return ok, nil
}

func (r *inMemoryUserRepo) UpdateProfile(ctx context.Context, id uuid.UUID, nickname string, avatarURL *string) (*model.User, error) {
	r.updateProfileCalls++
	r.updatedNickname = nickname
	r.updatedAvatar = avatarURL
	for _, u := range r.users {
		if u.ID == id {
			u.Nickname = nickname
			u.AvatarURL = avatarURL
			return u, nil
		}
	}
	return nil, nil
}

// stubVerifier 固定返回 ok/err 的 Turnstile 校验器。
type stubVerifier struct {
	ok  bool
	err error
}

func (s stubVerifier) Verify(context.Context, string, string) (bool, error) {
	return s.ok, s.err
}

// stubLimiter 记录调用但不实际限制。
type stubLimiter struct {
	locked     bool
	failures   int
	resetCalls int
}

func (s *stubLimiter) IsLocked(ctx context.Context, email string) (bool, error) { return s.locked, nil }
func (s *stubLimiter) RecordFailure(ctx context.Context, email string) error {
	s.failures++
	return nil
}
func (s *stubLimiter) Reset(ctx context.Context, email string) error { s.resetCalls++; return nil }

func newAuthTestSvc(repo *inMemoryUserRepo, verifier turnstile.Verifier, limiter LoginLimiter) *AuthService {
	jwtManager := auth.NewJWTManager("test-secret", time.Hour)
	return NewAuthService(repo, jwtManager, verifier, limiter)
}

func TestAuthService_Register_CreatesUser(t *testing.T) {
	repo := newInMemoryUserRepo()
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	user, token, err := svc.Register(context.Background(), "a@b.c", "password123", "cf-token")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, "a@b.c", user.Email)
	assert.NotEqual(t, "password123", user.HashedPassword)
	assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte("password123")))
	assert.NotEmpty(t, token)
}

func TestAuthService_Register_EmailTaken(t *testing.T) {
	repo := newInMemoryUserRepo()
	repo.users["a@b.c"] = &model.User{ID: uuid.New(), Email: "a@b.c"}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	_, _, err := svc.Register(context.Background(), "a@b.c", "password123", "cf-token")
	require.ErrorIs(t, err, ErrEmailTaken)
}

func TestAuthService_Register_CaptchaFailed(t *testing.T) {
	repo := newInMemoryUserRepo()
	svc := newAuthTestSvc(repo, stubVerifier{ok: false}, &stubLimiter{})

	_, _, err := svc.Register(context.Background(), "a@b.c", "password123", "bad-token")
	require.ErrorIs(t, err, ErrCaptchaFailed)
	assert.Len(t, repo.users, 0)
}

func TestAuthService_Login_Success(t *testing.T) {
	repo := newInMemoryUserRepo()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.users["a@b.c"] = &model.User{ID: uuid.New(), Email: "a@b.c", HashedPassword: string(hashed)}
	limiter := &stubLimiter{}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, limiter)

	user, token, err := svc.Login(context.Background(), "a@b.c", "password123", "cf-token")
	require.NoError(t, err)
	assert.Equal(t, "a@b.c", user.Email)
	assert.NotEmpty(t, token)
	assert.Equal(t, 1, limiter.resetCalls)
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	repo := newInMemoryUserRepo()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.users["a@b.c"] = &model.User{ID: uuid.New(), Email: "a@b.c", HashedPassword: string(hashed)}
	limiter := &stubLimiter{}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, limiter)

	_, _, err := svc.Login(context.Background(), "a@b.c", "wrong-password", "cf-token")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Equal(t, 1, limiter.failures)
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	repo := newInMemoryUserRepo()
	limiter := &stubLimiter{}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, limiter)

	_, _, err := svc.Login(context.Background(), "ghost@b.c", "password123", "cf-token")
	require.ErrorIs(t, err, ErrInvalidCredentials)
	assert.Equal(t, 1, limiter.failures)
}

func TestAuthService_Login_Locked(t *testing.T) {
	repo := newInMemoryUserRepo()
	limiter := &stubLimiter{locked: true}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, limiter)

	_, _, err := svc.Login(context.Background(), "a@b.c", "password123", "cf-token")
	require.ErrorIs(t, err, ErrTooManyAttempts)
	assert.Equal(t, 0, limiter.failures)
}

func TestAuthService_Register_NormalizesEmail(t *testing.T) {
	repo := newInMemoryUserRepo()
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	user, _, err := svc.Register(context.Background(), "  A@B.C ", "password123", "cf-token")
	require.NoError(t, err)
	assert.Equal(t, "a@b.c", user.Email)
	assert.Equal(t, "a@b.c", repo.users["a@b.c"].Email)
	assert.NotNil(t, repo.users["a@b.c"])
}

func TestAuthService_Login_EmailNormalized(t *testing.T) {
	repo := newInMemoryUserRepo()
	hashed, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	repo.users["a@b.c"] = &model.User{ID: uuid.New(), Email: "a@b.c", HashedPassword: string(hashed)}
	limiter := &stubLimiter{}
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, limiter)

	user, token, err := svc.Login(context.Background(), " A@B.C ", "password123", "cf-token")
	require.NoError(t, err)
	assert.Equal(t, "a@b.c", user.Email)
	assert.NotEmpty(t, token)
	assert.Equal(t, 1, limiter.resetCalls)
}

func TestAuthService_Register_CaptchaVerifierError(t *testing.T) {
	repo := newInMemoryUserRepo()
	svc := newAuthTestSvc(repo, stubVerifier{ok: false, err: errors.New("boom")}, &stubLimiter{})

	_, _, err := svc.Register(context.Background(), "a@b.c", "password123", "bad-token")
	require.ErrorIs(t, err, ErrCaptchaFailed)
	assert.Len(t, repo.users, 0)
}

func TestAuthService_UpdateProfileTrimsAndRejectsEmptyNickname(t *testing.T) {
	repo := newInMemoryUserRepo()
	user := &model.User{ID: uuid.New(), Email: "a@b.c", Nickname: "旧昵称"}
	repo.users[user.Email] = user
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	_, err := svc.UpdateProfile(context.Background(), uuid.New(), "   ", nil)
	assert.ErrorIs(t, err, ErrProfileInvalid)
	assert.Equal(t, 0, repo.updateProfileCalls)
}

func TestAuthService_UpdateProfileTrimsNicknameAndEmptyAvatar(t *testing.T) {
	repo := newInMemoryUserRepo()
	user := &model.User{ID: uuid.New(), Email: "a@b.c", Nickname: "旧昵称"}
	repo.users[user.Email] = user
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	empty := "  "
	updated, err := svc.UpdateProfile(context.Background(), user.ID, "  新昵称  ", &empty)
	require.NoError(t, err)
	assert.Equal(t, "新昵称", updated.Nickname)
	assert.Nil(t, updated.AvatarURL)
}

func TestAuthService_UpdateProfileRejectsTooLongNickname(t *testing.T) {
	repo := newInMemoryUserRepo()
	user := &model.User{ID: uuid.New(), Email: "a@b.c"}
	repo.users[user.Email] = user
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	_, err := svc.UpdateProfile(context.Background(), uuid.New(), strings.Repeat("很", 51), nil)
	assert.ErrorIs(t, err, ErrProfileInvalid)
}

func TestAuthService_UpdateProfileUserNotFound(t *testing.T) {
	repo := newInMemoryUserRepo()
	svc := newAuthTestSvc(repo, stubVerifier{ok: true}, &stubLimiter{})

	_, err := svc.UpdateProfile(context.Background(), uuid.New(), "新昵称", nil)
	assert.ErrorIs(t, err, ErrUserNotFound)
}
