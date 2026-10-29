// Package bilibili 封装 B 站公开 API：视频信息、字幕与评论获取。
package bilibili

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	// DefaultBaseURL 是 B 站公开 API 的地址。
	DefaultBaseURL = "https://api.bilibili.com"

	userAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"
)

// VideoInfo 是一个 B 站视频的聚合信息。
type VideoInfo struct {
	AID         int64
	BVID        string
	CID         int64 // 默认取第一 P
	Title       string
	Description string
	OwnerName   string
	CoverURL    string
	Duration    int // 秒
	Pages       []VideoPage
	Subtitles   []SubtitleMeta
}

// VideoPage 是分 P 信息。
type VideoPage struct {
	CID      int64
	Page     int
	Part     string
	Duration int
}

// SubtitleMeta 是一条字幕轨的元信息。
type SubtitleMeta struct {
	ID          int64
	Lan         string
	LanDoc      string
	SubtitleURL string // 可能为空，需要走 wbi/v2 降级解析
	AIStatus    int    // 1 表示 AI 生成字幕
}

// SubtitleSegment 是一条带时间轴的字幕文本（时间单位：秒）。
type SubtitleSegment struct {
	From    float64
	To      float64
	Content string
}

// CommentItem 是一条评论。
type CommentItem struct {
	RPID    int64
	Message string
	Like    int
	IsTop   bool
	Member  string
}

// subtitleFile 对应 B 站字幕 JSON 文件结构。
type subtitleFile struct {
	Body []struct {
		From    float64 `json:"from"`
		To      float64 `json:"to"`
		Content string  `json:"content"`
	} `json:"body"`
}

// Client 是 B 站 API 客户端。
type Client struct {
	httpClient *http.Client
	baseURL    string
	cookie     string
	logger     *slog.Logger
	// wbi 在副本间共享，WithCookie 后仍复用同一份密钥缓存。
	wbi *wbiSigner
}

// ErrLoginRequired 表示该视频的字幕需要 B 站登录态才能读取。
var ErrLoginRequired = errors.New("bilibili login required for subtitle")

// New 创建 B 站客户端。baseURL 可传 DefaultBaseURL；logger 为 nil 时使用默认日志。
func New(baseURL string, logger *slog.Logger) *Client {
	return NewWithCookie(baseURL, "", logger)
}

// NewWithCookie 创建携带服务端 B 站登录态的客户端。
func NewWithCookie(baseURL, cookie string, logger *slog.Logger) *Client {
	return newClient(baseURL, cookie, &http.Client{Timeout: 15 * time.Second}, logger)
}

// NewWithHTTPClient 创建使用自定义 http.Client 的 B 站客户端（便于测试注入）。
func NewWithHTTPClient(baseURL string, httpClient *http.Client, logger *slog.Logger) *Client {
	return newClient(baseURL, "", httpClient, logger)
}

func newClient(baseURL, cookie string, httpClient *http.Client, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	return &Client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(baseURL, "/"),
		cookie:     strings.TrimSpace(cookie),
		logger:     logger,
		wbi:        &wbiSigner{},
	}
}

// WithCookie 返回携带指定 Cookie 的副本；cookie 为空时返回原客户端。
// 副本与原客户端共享 http.Client 与 wbi 密钥缓存。
func (c *Client) WithCookie(cookie string) *Client {
	cookie = strings.TrimSpace(cookie)
	if cookie == "" || cookie == c.cookie {
		return c
	}
	clone := *c
	clone.cookie = cookie
	return &clone
}

// hasCookie 判断客户端是否携带登录态。
func (c *Client) hasCookie() bool {
	return c.cookie != ""
}

// get 发起带 UA 的 GET 请求（B 站接口无 UA 会被风控拦截）。
func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://www.bilibili.com")
	if c.cookie != "" {
		req.Header.Set("Cookie", c.cookie)
	}
	return c.httpClient.Do(req)
}

// getNoCookie 用于字幕 CDN 等第三方域名：只带 UA/Referer，不泄露登录态。
func (c *Client) getNoCookie(ctx context.Context, rawURL string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Referer", "https://www.bilibili.com")
	return c.httpClient.Do(req)
}

// GetVideoInfo 获取视频基础信息、分 P 与字幕轨列表。
func (c *Client) GetVideoInfo(ctx context.Context, bvid string) (*VideoInfo, error) {
	apiURL := fmt.Sprintf("%s/x/web-interface/view?bvid=%s", c.baseURL, bvid)

	resp, err := c.get(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("请求视频信息失败: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			AID      int64  `json:"aid"`
			BVID     string `json:"bvid"`
			Title    string `json:"title"`
			Desc     string `json:"desc"`
			Duration int    `json:"duration"`
			Pic      string `json:"pic"`
			Owner    struct {
				Name string `json:"name"`
			} `json:"owner"`
			Pages []struct {
				CID      int64  `json:"cid"`
				Page     int    `json:"page"`
				Part     string `json:"part"`
				Duration int    `json:"duration"`
			} `json:"pages"`
			Subtitle struct {
				List []struct {
					ID          int64  `json:"id"`
					Lan         string `json:"lan"`
					LanDoc      string `json:"lan_doc"`
					SubtitleURL string `json:"subtitle_url"`
					AIStatus    int    `json:"ai_status"`
				} `json:"list"`
			} `json:"subtitle"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析视频信息响应失败: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("B站API错误: code=%d, msg=%s", result.Code, result.Message)
	}

	info := &VideoInfo{
		AID:         result.Data.AID,
		BVID:        result.Data.BVID,
		Title:       result.Data.Title,
		Description: result.Data.Desc,
		OwnerName:   result.Data.Owner.Name,
		// B站图床同时支持 http/https；统一存 https，
		// 否则 Android 端 networkSecurityConfig 拦截明文流量导致封面加载失败
		CoverURL: strings.Replace(result.Data.Pic, "http://", "https://", 1),
		Duration: result.Data.Duration,
	}
	for _, p := range result.Data.Pages {
		info.Pages = append(info.Pages, VideoPage{
			CID:      p.CID,
			Page:     p.Page,
			Part:     p.Part,
			Duration: p.Duration,
		})
	}
	if len(info.Pages) > 0 {
		info.CID = info.Pages[0].CID
	}
	for _, s := range result.Data.Subtitle.List {
		info.Subtitles = append(info.Subtitles, SubtitleMeta{
			ID:          s.ID,
			Lan:         s.Lan,
			LanDoc:      s.LanDoc,
			SubtitleURL: s.SubtitleURL,
			AIStatus:    s.AIStatus,
		})
	}

	c.logger.Info("获取视频信息成功",
		slog.String("bvid", bvid),
		slog.String("title", info.Title),
		slog.Int("subtitle_tracks", len(info.Subtitles)))
	return info, nil
}

// FetchBestSubtitle 选取最优字幕轨并拉取字幕内容；无字幕时返回 (nil, false, nil)。
// 视频字幕仅登录可见且当前客户端没有登录态时返回 ErrLoginRequired。
func (c *Client) FetchBestSubtitle(ctx context.Context, info *VideoInfo) ([]SubtitleSegment, bool, error) {
	if info.CID == 0 {
		return nil, false, nil
	}

	tracks := info.Subtitles
	if len(tracks) == 0 {
		// view 接口匿名访问经常返回空列表，需要查 player 接口确认是真没字幕还是仅登录可见
		player, err := c.fetchPlayerSubtitles(ctx, info)
		if err != nil {
			return nil, false, err
		}
		if player.NeedLogin && !c.hasCookie() {
			return nil, false, ErrLoginRequired
		}
		tracks = player.Subtitles
		if len(tracks) == 0 {
			c.logger.Info("视频无字幕", slog.String("bvid", info.BVID))
			return nil, false, nil
		}
	}

	track := selectBestSubtitle(tracks)

	// 新版 view 接口经常返回空的 subtitle_url，需要走 wbi/v2 降级拿真实地址
	subtitleURL := track.SubtitleURL
	if subtitleURL == "" {
		player, err := c.fetchPlayerSubtitles(ctx, info)
		if err != nil {
			return nil, false, fmt.Errorf("解析字幕地址失败: %w", err)
		}
		if player.NeedLogin && !c.hasCookie() {
			return nil, false, ErrLoginRequired
		}
		for _, s := range player.Subtitles {
			if s.Lan == track.Lan && s.SubtitleURL != "" {
				subtitleURL = s.SubtitleURL
				break
			}
		}
		if subtitleURL == "" {
			return nil, false, fmt.Errorf("wbi/v2 中未找到 lan=%s 的字幕", track.Lan)
		}
	}

	segments, err := c.fetchSubtitleContent(ctx, subtitleURL)
	if err != nil {
		return nil, false, fmt.Errorf("获取字幕内容失败: %w", err)
	}
	if len(segments) == 0 {
		return nil, false, nil
	}

	c.logger.Info("获取字幕成功",
		slog.String("bvid", info.BVID),
		slog.String("lan", track.Lan),
		slog.Int("segments", len(segments)))
	return segments, true, nil
}

// selectBestSubtitle 按语言与来源优先级选择字幕轨：简体中文人工字幕优先，AI 字幕降权。
func selectBestSubtitle(tracks []SubtitleMeta) SubtitleMeta {
	priority := map[string]int{
		"zh-CN":   100, // 简体中文（人工）
		"zh-Hans": 95,
		"zh":      90,
		"ai-zh":   80, // AI 中文
		"zh-TW":   70,
		"zh-Hant": 65,
	}

	scored := make([]SubtitleMeta, len(tracks))
	copy(scored, tracks)
	sort.SliceStable(scored, func(i, j int) bool {
		return subtitleScore(scored[i], priority) > subtitleScore(scored[j], priority)
	})
	return scored[0]
}

func subtitleScore(s SubtitleMeta, priority map[string]int) int {
	score, ok := priority[s.Lan]
	if !ok {
		score = 10
	}
	if s.AIStatus == 1 {
		score -= 20
	}
	return score
}

// playerSubtitle 是 /x/player/wbi/v2 返回的字幕信息。
type playerSubtitle struct {
	Subtitles []SubtitleMeta
	// NeedLogin 为 true 表示该视频的字幕需要登录后才能读取。
	NeedLogin bool
}

// fetchPlayerSubtitles 调 /x/player/wbi/v2 获取字幕轨（带 wbi 签名）。
func (c *Client) fetchPlayerSubtitles(ctx context.Context, info *VideoInfo) (*playerSubtitle, error) {
	if info.BVID == "" || info.CID == 0 {
		return nil, fmt.Errorf("缺少必要参数: bvid=%s cid=%d", info.BVID, info.CID)
	}

	params := url.Values{
		"bvid": {info.BVID},
		"cid":  {fmt.Sprintf("%d", info.CID)},
	}
	query, err := c.signedQuery(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("生成 wbi 签名失败: %w", err)
	}

	result, err := c.requestPlayerSubtitles(ctx, query)
	if err != nil {
		return nil, err
	}
	// 密钥过期时 B 站返回 -403，刷新密钥后重试一次
	if result.Code == -403 {
		c.invalidateKeys()
		query, err = c.signedQuery(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("刷新 wbi 签名失败: %w", err)
		}
		result, err = c.requestPlayerSubtitles(ctx, query)
		if err != nil {
			return nil, err
		}
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("wbi/v2 错误: code=%d msg=%s", result.Code, result.Message)
	}

	out := &playerSubtitle{NeedLogin: result.Data.NeedLoginSubtitle}
	for _, s := range result.Data.Subtitle.Subtitles {
		if s.SubtitleURL == "" {
			continue
		}
		out.Subtitles = append(out.Subtitles, SubtitleMeta{
			Lan:         s.Lan,
			LanDoc:      s.LanDoc,
			SubtitleURL: s.SubtitleURL,
		})
	}
	return out, nil
}

type playerSubtitleResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		NeedLoginSubtitle bool `json:"need_login_subtitle"`
		Subtitle          struct {
			Subtitles []struct {
				Lan         string `json:"lan"`
				LanDoc      string `json:"lan_doc"`
				SubtitleURL string `json:"subtitle_url"`
			} `json:"subtitles"`
		} `json:"subtitle"`
	} `json:"data"`
}

func (c *Client) requestPlayerSubtitles(ctx context.Context, query string) (*playerSubtitleResponse, error) {
	apiURL := fmt.Sprintf("%s/x/player/wbi/v2?%s", c.baseURL, query)
	resp, err := c.get(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("请求 wbi/v2 失败: %w", err)
	}
	defer resp.Body.Close()

	var result playerSubtitleResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("解析 wbi/v2 响应失败: %w", err)
	}
	return &result, nil
}

// fetchSubtitleContent 下载字幕 JSON 并解析为时间轴分段。
func (c *Client) fetchSubtitleContent(ctx context.Context, subtitleURL string) ([]SubtitleSegment, error) {
	// 处理协议相对路径（如 //i0.hdslb.com/...）
	if strings.HasPrefix(subtitleURL, "//") {
		subtitleURL = "https:" + subtitleURL
	}

	resp, err := c.getNoCookie(ctx, subtitleURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var file subtitleFile
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return nil, fmt.Errorf("解析字幕文件失败: %w", err)
	}

	segments := make([]SubtitleSegment, 0, len(file.Body))
	for _, seg := range file.Body {
		text := strings.TrimSpace(seg.Content)
		if text == "" {
			continue
		}
		segments = append(segments, SubtitleSegment{From: seg.From, To: seg.To, Content: text})
	}
	return segments, nil
}

// GetRecipeLikeComments 拉取置顶评论与疑似包含菜谱的高赞评论（作为无字幕时的兜底文本来源）。
func (c *Client) GetRecipeLikeComments(ctx context.Context, aid int64, limit int) ([]CommentItem, error) {
	if aid == 0 {
		return nil, fmt.Errorf("aid为空")
	}

	apiURL := fmt.Sprintf("%s/x/v2/reply/main?type=1&oid=%d&ps=20", c.baseURL, aid)

	resp, err := c.get(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("请求评论失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取评论响应失败: %w", err)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			Top struct {
				Upper struct {
					RPID    int64 `json:"rpid"`
					Like    int   `json:"like"`
					Content struct {
						Message string `json:"message"`
					} `json:"content"`
					Member struct {
						Uname string `json:"uname"`
					} `json:"member"`
				} `json:"upper"`
			} `json:"top"`
			Replies []struct {
				RPID    int64 `json:"rpid"`
				Like    int   `json:"like"`
				Content struct {
					Message string `json:"message"`
				} `json:"content"`
				Member struct {
					Uname string `json:"uname"`
				} `json:"member"`
			} `json:"replies"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析评论响应失败: %w", err)
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("B站评论API错误: code=%d, msg=%s", result.Code, result.Message)
	}

	var comments []CommentItem
	if top := result.Data.Top.Upper; top.Content.Message != "" {
		comments = append(comments, CommentItem{
			RPID:    top.RPID,
			Message: top.Content.Message,
			Like:    top.Like,
			IsTop:   true,
			Member:  top.Member.Uname,
		})
	}
	for _, reply := range result.Data.Replies {
		if !LooksLikeRecipeComment(reply.Content.Message) {
			continue
		}
		comments = append(comments, CommentItem{
			RPID:    reply.RPID,
			Message: reply.Content.Message,
			Like:    reply.Like,
			IsTop:   false,
			Member:  reply.Member.Uname,
		})
		if len(comments) >= limit {
			break
		}
	}

	c.logger.Info("获取评论成功", slog.Int64("aid", aid), slog.Int("count", len(comments)))
	return comments, nil
}

// LooksLikeRecipeComment 通过关键词粗判一条评论是否像菜谱。
func LooksLikeRecipeComment(msg string) bool {
	keywords := []string{
		"食材", "材料", "配方", "步骤", "做法",
		"克", "勺", "分钟", "腌", "焯水",
		"爆香", "小火", "大火", "生抽", "老抽",
		"蚝油", "盐", "糖",
	}

	count := 0
	for _, kw := range keywords {
		if strings.Contains(msg, kw) {
			count++
		}
	}
	return count >= 2
}
