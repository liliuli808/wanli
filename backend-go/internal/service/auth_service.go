package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/wanli/backend-go/internal/auth"
	"github.com/wanli/backend-go/internal/infrastructure/turnstile"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/internal/repository"
)

var (
	ErrProfileInvalid     = errors.New("昵称不能为空且不超过 50 字")
	ErrUserNotFound       = errors.New("用户不存在")
	ErrEmailTaken         = errors.New("该邮箱已被注册")
	ErrInvalidCredentials = errors.New("邮箱或密码错误")
	ErrCaptchaFailed      = errors.New("人机验证未通过，请重试")
	ErrTooManyAttempts    = errors.New("失败次数过多，请稍后再试")
)

// dummyPasswordHash 用于账号不存在时的假 bcrypt 比较，抹平时序差异，防止通过响应时间枚举账号。
var dummyPasswordHash = []byte("$2a$10$7EqJtq98hPqEX7fNZaFWoOhi7HyMZPOnZ5rOk3R0KvCqK0g9J8r7W")

// AuthService 处理认证相关业务。
type AuthService struct {
	userRepo     repository.UserRepository
	jwtManager   *auth.JWTManager
	turnstile    turnstile.Verifier
	loginLimiter LoginLimiter
}

// NewAuthService 创建 AuthService。
func NewAuthService(
	userRepo repository.UserRepository,
	jwtManager *auth.JWTManager,
	turnstile turnstile.Verifier,
	loginLimiter LoginLimiter,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		jwtManager:   jwtManager,
		turnstile:    turnstile,
		loginLimiter: loginLimiter,
	}
}

// Register 创建新账号并签发 token。
func (s *AuthService) Register(ctx context.Context, email, password, turnstileToken string) (*model.User, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	ok, err := s.turnstile.Verify(ctx, turnstileToken, "")
	if err != nil || !ok {
		return nil, "", ErrCaptchaFailed
	}

	exists, err := s.userRepo.EmailExists(ctx, email)
	if err != nil {
		return nil, "", fmt.Errorf("check email exists: %w", err)
	}
	if exists {
		return nil, "", ErrEmailTaken
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		ID:             uuid.New(),
		Email:          email,
		HashedPassword: string(hashed),
		Nickname:       generateNickname(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	token, err := s.jwtManager.Generate(user.ID.String(), user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	return user, token, nil
}

// Login 校验人机验证、限流与密码，签发 token。
func (s *AuthService) Login(ctx context.Context, email, password, turnstileToken string) (*model.User, string, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	ok, err := s.turnstile.Verify(ctx, turnstileToken, "")
	if err != nil || !ok {
		return nil, "", ErrCaptchaFailed
	}

	locked, err := s.loginLimiter.IsLocked(ctx, email)
	if err != nil {
		return nil, "", fmt.Errorf("check login limit: %w", err)
	}
	if locked {
		return nil, "", ErrTooManyAttempts
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", fmt.Errorf("get user: %w", err)
	}
	if user == nil {
		// 假比较以均衡耗时，避免通过响应时间枚举账号是否存在。
		_ = bcrypt.CompareHashAndPassword(dummyPasswordHash, []byte(password))
		if err := s.loginLimiter.RecordFailure(ctx, email); err != nil {
			return nil, "", fmt.Errorf("record login failure: %w", err)
		}
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(password)); err != nil {
		if err := s.loginLimiter.RecordFailure(ctx, email); err != nil {
			return nil, "", fmt.Errorf("record login failure: %w", err)
		}
		return nil, "", ErrInvalidCredentials
	}

	if err := s.loginLimiter.Reset(ctx, email); err != nil {
		return nil, "", fmt.Errorf("reset login failures: %w", err)
	}

	token, err := s.jwtManager.Generate(user.ID.String(), user.Email)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}
	return user, token, nil
}

func generateNickname() string {
	return "碗友_" + uuid.New().String()[:8]
}

// GetProfile 返回用户完整资料；用户不存在返回 ErrUserNotFound。
func (s *AuthService) GetProfile(ctx context.Context, userID uuid.UUID) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// UpdateProfile 校验并更新昵称/头像。昵称去首尾空白后需为 1~50 字；头像空白串视为清除。
func (s *AuthService) UpdateProfile(ctx context.Context, userID uuid.UUID, nickname string, avatarURL *string) (*model.User, error) {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" || len([]rune(nickname)) > 50 {
		return nil, ErrProfileInvalid
	}
	if avatarURL != nil {
		trimmed := strings.TrimSpace(*avatarURL)
		if trimmed == "" {
			avatarURL = nil
		} else {
			avatarURL = &trimmed
		}
	}
	user, err := s.userRepo.UpdateProfile(ctx, userID, nickname, avatarURL)
	if err != nil {
		return nil, fmt.Errorf("update profile: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}
