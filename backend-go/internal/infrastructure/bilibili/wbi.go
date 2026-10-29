package bilibili

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// mixinKeyEncTab 是 B 站 wbi 签名的固定混淆表。
var mixinKeyEncTab = []int{
	46, 47, 18, 2, 53, 8, 23, 32, 15, 50, 10, 31, 58, 3, 45, 35, 27, 43, 5, 49,
	33, 9, 42, 19, 29, 28, 14, 39, 12, 38, 41, 13, 37, 48, 7, 16, 24, 55, 40,
	61, 26, 17, 0, 1, 60, 51, 30, 4, 22, 25, 54, 21, 56, 59, 6, 63, 57, 62, 11,
	36, 20, 34, 44, 52,
}

// wbiKeyTTL 是 wbi 密钥的缓存时长。B 站每天轮换密钥，1 小时足够保守。
const wbiKeyTTL = time.Hour

// wbiSigner 缓存 img_key/sub_key，避免每次请求都调 nav 接口。
// 通过指针在 Client 副本间共享，WithCookie 不会导致缓存失效。
type wbiSigner struct {
	mu        sync.Mutex
	imgKey    string
	subKey    string
	fetchedAt time.Time
}

// mixinKey 按混淆表从 img_key + sub_key 派生 32 位签名密钥。
func mixinKey(imgKey, subKey string) string {
	raw := imgKey + subKey
	var b strings.Builder
	for _, idx := range mixinKeyEncTab {
		if idx >= len(raw) {
			continue
		}
		b.WriteByte(raw[idx])
		if b.Len() == 32 {
			break
		}
	}
	return b.String()
}

// sign 为参数追加 wts 与 w_rid，返回可直接拼接的 query 串。
func sign(params url.Values, key string, now time.Time) string {
	params.Set("wts", strconv.FormatInt(now.Unix(), 10))

	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(url.QueryEscape(k))
		b.WriteByte('=')
		// B 站要求过滤 !'()* 后再做百分号编码
		b.WriteString(url.QueryEscape(filterWbiValue(params.Get(k))))
	}
	query := b.String()

	sum := md5.Sum([]byte(query + key))
	return query + "&w_rid=" + hex.EncodeToString(sum[:])
}

// filterWbiValue 去掉 B 站签名规范里要求过滤的字符。
func filterWbiValue(v string) string {
	return strings.NewReplacer("!", "", "'", "", "(", "", ")", "", "*", "").Replace(v)
}

// keyFilename 从 wbi 图片 URL 中取出文件名（不含扩展名）作为密钥。
func keyFilename(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if i := strings.LastIndex(rawURL, "/"); i >= 0 {
		rawURL = rawURL[i+1:]
	}
	if i := strings.Index(rawURL, "."); i >= 0 {
		rawURL = rawURL[:i]
	}
	return rawURL
}

// getKeys 返回缓存的 wbi 密钥，过期或为空时重新拉取。
func (c *Client) getKeys(ctx context.Context) (string, string, error) {
	s := c.wbi
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.imgKey != "" && s.subKey != "" && time.Since(s.fetchedAt) < wbiKeyTTL {
		return s.imgKey, s.subKey, nil
	}

	imgKey, subKey, err := c.fetchKeys(ctx)
	if err != nil {
		return "", "", err
	}
	s.imgKey, s.subKey, s.fetchedAt = imgKey, subKey, time.Now()
	return imgKey, subKey, nil
}

// invalidateKeys 丢弃缓存，下次调用重新拉取（签名被拒时使用）。
func (c *Client) invalidateKeys() {
	c.wbi.mu.Lock()
	c.wbi.imgKey, c.wbi.subKey, c.wbi.fetchedAt = "", "", time.Time{}
	c.wbi.mu.Unlock()
}

// fetchKeys 调 nav 接口拿 wbi 图片地址。未登录时 nav 返回 code=-101，
// 但 data.wbi_img 仍然有效，因此不按 code 判定失败。
func (c *Client) fetchKeys(ctx context.Context) (string, string, error) {
	resp, err := c.get(ctx, c.baseURL+"/x/web-interface/nav")
	if err != nil {
		return "", "", fmt.Errorf("请求 wbi 密钥失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code int `json:"code"`
		Data struct {
			WbiImg struct {
				ImgURL string `json:"img_url"`
				SubURL string `json:"sub_url"`
			} `json:"wbi_img"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", fmt.Errorf("解析 wbi 密钥响应失败: %w", err)
	}

	imgKey := keyFilename(result.Data.WbiImg.ImgURL)
	subKey := keyFilename(result.Data.WbiImg.SubURL)
	if imgKey == "" || subKey == "" {
		return "", "", fmt.Errorf("nav 未返回有效 wbi 密钥: code=%d", result.Code)
	}
	return imgKey, subKey, nil
}

// signedQuery 为一次请求生成带 wts/w_rid 的 query 串。
func (c *Client) signedQuery(ctx context.Context, params url.Values) (string, error) {
	imgKey, subKey, err := c.getKeys(ctx)
	if err != nil {
		return "", err
	}
	return sign(params, mixinKey(imgKey, subKey), time.Now()), nil
}
