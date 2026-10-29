package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
)

// NavInfo 是 nav 接口返回的账号信息。
type NavInfo struct {
	// IsLogin 为 false 表示 Cookie 缺失、无效或已过期（B 站返回 code=-101）。
	IsLogin bool
	Mid     int64
	Uname   string
}

// Nav 查询当前 Cookie 对应的账号信息，用于校验用户授权的 SESSDATA 是否有效。
func (c *Client) Nav(ctx context.Context) (*NavInfo, error) {
	resp, err := c.get(ctx, c.baseURL+"/x/web-interface/nav")
	if err != nil {
		return nil, fmt.Errorf("请求 nav 失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
		Data struct {
			IsLogin bool   `json:"isLogin"`
			Mid     int64  `json:"mid"`
			Uname   string `json:"uname"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 nav 响应失败: %w", err)
	}

	// -101 表示未登录，属于可预期的“凭证无效”，不算接口故障
	if result.Code == -101 {
		return &NavInfo{IsLogin: false}, nil
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("nav 错误: code=%d msg=%s", result.Code, result.Msg)
	}
	if !result.Data.IsLogin {
		return &NavInfo{IsLogin: false}, nil
	}
	return &NavInfo{IsLogin: true, Mid: result.Data.Mid, Uname: result.Data.Uname}, nil
}
