// Package parser 提供各类外部输入的解析工具。
package parser

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// BVID 固定为 "BV" 前缀 + 10 位字母数字。
var bvidRegex = regexp.MustCompile(`BV[0-9A-Za-z]{10}`)

// ParseBVID 从用户输入（完整链接、短链或纯 BV 号）中提取 BV 号。
func ParseBVID(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", errors.New("输入为空")
	}

	// 短链需要先跳转还原成真实链接
	if strings.Contains(input, "b23.tv") || strings.Contains(input, "bili.im") {
		resolved, err := ResolveShortURL(input)
		if err != nil {
			return "", fmt.Errorf("短链解析失败: %w", err)
		}
		input = resolved
	}

	bvid := bvidRegex.FindString(input)
	if bvid == "" {
		return "", errors.New("未识别到BV号")
	}
	return bvid, nil
}

// ResolveShortURL 跟随 B 站短链的一次 302 跳转，返回真实链接。
func ResolveShortURL(rawURL string) (string, error) {
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "https://" + rawURL
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 不自动跳转，直接取 Location 头
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if location := resp.Header.Get("Location"); location != "" {
		return location, nil
	}
	return "", errors.New("无法解析短链跳转")
}

// IsValidBilibiliURL 粗判输入是否可能是 B 站视频链接或 BV 号。
func IsValidBilibiliURL(input string) bool {
	if bvidRegex.MatchString(input) {
		return true
	}
	input = strings.ToLower(strings.TrimSpace(input))
	return strings.Contains(input, "bilibili.com") ||
		strings.Contains(input, "b23.tv") ||
		strings.Contains(input, "bili.im")
}
