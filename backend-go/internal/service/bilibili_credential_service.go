package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/wanli/backend-go/internal/infrastructure/bilibili"
	"github.com/wanli/backend-go/internal/model"
	"github.com/wanli/backend-go/pkg/secretbox"
)

var (
	// ErrBilibiliCredentialInvalid 表示用户提交的 SESSDATA 无法登录 B 站（无效或已过期）。
	ErrBilibiliCredentialInvalid = errors.New("bilibili credential invalid")
	// ErrBilibiliCredentialDisabled 表示服务端没有配置凭证加密密钥，授权功能不可用。
	ErrBilibiliCredentialDisabled = errors.New("bilibili credential storage disabled")
)

// sessdataRegex 限定 SESSDATA 可接受的字符集，避免 Cookie 注入或异常超长内容落库。
var sessdataRegex = regexp.MustCompile(`^[A-Za-z0-9%,*._\-]+$`)

const (
	minSessdataLen = 10
	maxSessdataLen = 512
)

// BilibiliCredentialStore 抽象用户凭证的持久化。
type BilibiliCredentialStore interface {
	GetBilibiliCredential(ctx context.Context, id uuid.UUID) (*model.BilibiliCredential, error)
	SetBilibiliCredential(ctx context.Context, id uuid.UUID, encSessdata, mid, uname string) error
	ClearBilibiliCredential(ctx context.Context, id uuid.UUID) error
}

// BilibiliCredentialStatus 是返回给客户端的授权状态（不含任何凭证内容）。
type BilibiliCredentialStatus struct {
	Bound     bool
	Mid       string
	Uname     string
	UpdatedAt *time.Time
}

// BilibiliCredentialService 管理用户授权的 B 站登录态。
type BilibiliCredentialService struct {
	store  BilibiliCredentialStore
	box    *secretbox.Box
	bili   *bilibili.Client
	logger *slog.Logger
}

// NewBilibiliCredentialService 创建服务。box 为 nil 时授权功能不可用。
func NewBilibiliCredentialService(
	store BilibiliCredentialStore,
	box *secretbox.Box,
	bili *bilibili.Client,
	logger *slog.Logger,
) *BilibiliCredentialService {
	if logger == nil {
		logger = slog.Default()
	}
	return &BilibiliCredentialService{store: store, box: box, bili: bili, logger: logger}
}

// Bind 校验并保存用户提交的 SESSDATA。
func (s *BilibiliCredentialService) Bind(ctx context.Context, userID uuid.UUID, sessdata string) (*BilibiliCredentialStatus, error) {
	if s.box == nil {
		return nil, ErrBilibiliCredentialDisabled
	}
	sessdata = normalizeSessdata(sessdata)
	if !isPlausibleSessdata(sessdata) {
		return nil, ErrBilibiliCredentialInvalid
	}

	nav, err := s.bili.WithCookie("SESSDATA=" + sessdata).Nav(ctx)
	if err != nil {
		return nil, fmt.Errorf("校验 B 站登录态失败: %w", err)
	}
	if !nav.IsLogin {
		return nil, ErrBilibiliCredentialInvalid
	}

	enc, err := s.box.Encrypt(sessdata)
	if err != nil {
		return nil, fmt.Errorf("加密凭证失败: %w", err)
	}
	mid := fmt.Sprintf("%d", nav.Mid)
	if err := s.store.SetBilibiliCredential(ctx, userID, enc, mid, nav.Uname); err != nil {
		return nil, err
	}

	s.logger.Info("用户已授权 B 站账号",
		slog.String("user_id", userID.String()),
		slog.String("bilibili_mid", mid))
	return s.Status(ctx, userID)
}

// Unbind 解除授权。
func (s *BilibiliCredentialService) Unbind(ctx context.Context, userID uuid.UUID) error {
	if s.box == nil {
		return ErrBilibiliCredentialDisabled
	}
	if err := s.store.ClearBilibiliCredential(ctx, userID); err != nil {
		return err
	}
	s.logger.Info("用户已解除 B 站授权", slog.String("user_id", userID.String()))
	return nil
}

// Status 返回授权状态；未授权时 Bound=false。
func (s *BilibiliCredentialService) Status(ctx context.Context, userID uuid.UUID) (*BilibiliCredentialStatus, error) {
	cred, err := s.store.GetBilibiliCredential(ctx, userID)
	if err != nil {
		return nil, err
	}
	if cred == nil {
		return &BilibiliCredentialStatus{Bound: false}, nil
	}
	return &BilibiliCredentialStatus{
		Bound:     true,
		Mid:       cred.Mid,
		Uname:     cred.Uname,
		UpdatedAt: cred.UpdatedAt,
	}, nil
}

// Sessdata 返回解密后的 SESSDATA；未授权或解密失败时返回空串。
// 解密失败只记日志不报错：凭证不可用时按匿名访问降级，不影响主流程。
func (s *BilibiliCredentialService) Sessdata(ctx context.Context, userID uuid.UUID) string {
	if s.box == nil {
		return ""
	}
	cred, err := s.store.GetBilibiliCredential(ctx, userID)
	if err != nil {
		s.logger.Warn("读取 B 站凭证失败，按匿名访问处理",
			slog.String("user_id", userID.String()), slog.String("error", err.Error()))
		return ""
	}
	if cred == nil {
		return ""
	}
	sessdata, err := s.box.Decrypt(cred.EncSessdata)
	if err != nil {
		s.logger.Warn("解密 B 站凭证失败，按匿名访问处理",
			slog.String("user_id", userID.String()), slog.String("error", err.Error()))
		return ""
	}
	return sessdata
}

// normalizeSessdata 去掉用户复制时可能带入的前缀、空白与引号。
func normalizeSessdata(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "SESSDATA=")
	raw = strings.Trim(raw, `"'`)
	return strings.TrimSpace(raw)
}

// isPlausibleSessdata 做基本的格式校验；真正的有效性由 B 站 nav 接口判定。
func isPlausibleSessdata(s string) bool {
	if len(s) < minSessdataLen || len(s) > maxSessdataLen {
		return false
	}
	return sessdataRegex.MatchString(s)
}
