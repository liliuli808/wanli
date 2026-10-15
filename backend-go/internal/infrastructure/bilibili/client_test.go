package bilibili

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newFakeServer 返回一个模拟 B 站 API 的 httptest 服务器。
// view 接口返回的字幕轨 subtitle_url 为空（模拟新版 B 站 API 行为），
// 真实字幕地址需要通过 wbi/v2 降级解析获得。
func newFakeServer(t *testing.T) *httptest.Server {
	t.Helper()

	var server *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/x/web-interface/view", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"code": 0,
			"message": "0",
			"data": {
				"aid": 123456,
				"bvid": "BV1xx411c7xx",
				"title": "家常红烧肉",
				"desc": "今天做红烧肉，五花肉500克",
				"duration": 300,
				"pic": "http://i0.hdslb.com/cover.jpg",
				"owner": {"name": "美食作家"},
				"pages": [{"cid": 111, "page": 1, "part": "正片", "duration": 300}],
				"subtitle": {
					"list": [
						{"id": 1, "lan": "ai-zh", "lan_doc": "AI中文", "subtitle_url": "", "ai_status": 1},
						{"id": 2, "lan": "zh-CN", "lan_doc": "简体中文", "subtitle_url": "", "ai_status": 0}
					]
				}
			}
		}`)
	})
	mux.HandleFunc("/x/player/wbi/v2", func(w http.ResponseWriter, r *http.Request) {
		// zh-CN 故意返回协议相对路径，覆盖 URL 补全逻辑
		fmt.Fprintf(w, `{
			"code": 0,
			"message": "0",
			"data": {
				"subtitle": {
					"subtitles": [
						{"lan": "ai-zh", "subtitle_url": "%s/subtitle-ai.json"},
						{"lan": "zh-CN", "subtitle_url": "%s"}
					]
				}
			}
		}`, server.URL, protocolRelative(server.URL, "/subtitle-zh.json"))
	})
	mux.HandleFunc("/subtitle-zh.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"body": [
			{"from": 0, "to": 5, "content": "今天做一道家常红烧肉"},
			{"from": 6, "to": 12, "content": "准备五花肉500克"},
			{"from": 15, "to": 20, "content": "  "}
		]}`)
	})
	mux.HandleFunc("/subtitle-ai.json", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"body": [{"from": 0, "to": 5, "content": "AI字幕"}]}`)
	})
	mux.HandleFunc("/x/v2/reply/main", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{
			"code": 0,
			"message": "0",
			"data": {
				"top": {"upper": {
					"rpid": 1, "like": 100,
					"content": {"message": "文字版菜谱：五花肉500克，焯水后小火炖40分钟"},
					"member": {"uname": "课代表"}
				}},
				"replies": [
					{"rpid": 2, "like": 50, "content": {"message": "生抽两勺老抽一勺，盐少许，做法很简单"}, "member": {"uname": "路人甲"}},
					{"rpid": 3, "like": 30, "content": {"message": "看起来好好吃，收藏了"}, "member": {"uname": "路人乙"}},
					{"rpid": 4, "like": 10, "content": {"message": "配方：糖30克，腌10分钟再焯水"}, "member": {"uname": "路人丙"}}
				]
			}
		}`)
	})

	server = httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	return server
}

// newTestClient 创建指向测试服务器的客户端（TLS server 需注入其信任的 http.Client）。
func newTestClient(server *httptest.Server) *Client {
	return NewWithHTTPClient(server.URL, server.Client(), nil)
}

// protocolRelative 把绝对地址转成协议相对地址（如 //127.0.0.1:port/path）。
func protocolRelative(base, path string) string {
	return strings.TrimPrefix(base, "https:") + path
}

func TestGetVideoInfo(t *testing.T) {
	server := newFakeServer(t)
	client := newTestClient(server)

	info, err := client.GetVideoInfo(context.Background(), "BV1xx411c7xx")
	require.NoError(t, err)

	assert.Equal(t, int64(123456), info.AID)
	assert.Equal(t, "家常红烧肉", info.Title)
	assert.Equal(t, "美食作家", info.OwnerName)
	assert.Equal(t, int64(111), info.CID, "应默认取第一 P 的 CID")
	assert.Len(t, info.Subtitles, 2)
}

func TestGetVideoInfo_CookieHeader(t *testing.T) {
	tests := []struct {
		name   string
		cookie string
	}{
		{name: "authenticated", cookie: "SESSDATA=test-session; bili_jct=test-csrf"},
		{name: "guest", cookie: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var receivedCookie string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				receivedCookie = r.Header.Get("Cookie")
				fmt.Fprint(w, `{"code":0,"message":"0","data":{"bvid":"BV1xx411c7xx","pages":[],"subtitle":{"list":[]}}}`)
			}))
			t.Cleanup(server.Close)

			client := NewWithCookie(server.URL, tt.cookie, nil)
			_, err := client.GetVideoInfo(context.Background(), "BV1xx411c7xx")
			require.NoError(t, err)
			assert.Equal(t, tt.cookie, receivedCookie)
		})
	}
}

func TestGetVideoInfo_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"code": -404, "message": "啥都木有"}`)
	}))
	defer server.Close()

	client := newTestClient(server)
	_, err := client.GetVideoInfo(context.Background(), "BV1xx411c7xx")
	assert.ErrorContains(t, err, "-404")
}

func TestFetchBestSubtitle_Success(t *testing.T) {
	server := newFakeServer(t)
	client := newTestClient(server)

	info, err := client.GetVideoInfo(context.Background(), "BV1xx411c7xx")
	require.NoError(t, err)

	segments, ok, err := client.FetchBestSubtitle(context.Background(), info)
	require.NoError(t, err)
	require.True(t, ok)

	// 应通过 wbi/v2 降级拿到 zh-CN 人工字幕（AI 字幕降权），
	// 且协议相对路径已被补全、空白段被过滤
	require.Len(t, segments, 2)
	assert.Equal(t, "今天做一道家常红烧肉", segments[0].Content)
	assert.Equal(t, 0.0, segments[0].From)
	assert.Equal(t, 5.0, segments[0].To)
	assert.Equal(t, "准备五花肉500克", segments[1].Content)
}

func TestFetchBestSubtitle_NoSubtitle(t *testing.T) {
	client := New("http://unused", nil)
	info := &VideoInfo{BVID: "BV1xx411c7xx", CID: 111}

	segments, ok, err := client.FetchBestSubtitle(context.Background(), info)
	require.NoError(t, err)
	assert.False(t, ok)
	assert.Nil(t, segments)
}

func TestSelectBestSubtitle(t *testing.T) {
	tracks := []SubtitleMeta{
		{Lan: "ai-zh", AIStatus: 1},
		{Lan: "zh-TW"},
		{Lan: "zh-CN"},
	}
	best := selectBestSubtitle(tracks)
	assert.Equal(t, "zh-CN", best.Lan, "简体中文人工字幕应优先")

	// 只有 AI 字幕时也应能选出
	best = selectBestSubtitle([]SubtitleMeta{{Lan: "ai-zh", AIStatus: 1}})
	assert.Equal(t, "ai-zh", best.Lan)
}

func TestGetRecipeLikeComments(t *testing.T) {
	server := newFakeServer(t)
	client := newTestClient(server)

	comments, err := client.GetRecipeLikeComments(context.Background(), 123456, 5)
	require.NoError(t, err)

	require.Len(t, comments, 3)
	assert.True(t, comments[0].IsTop, "第一条应为置顶评论")
	assert.Equal(t, "课代表", comments[0].Member)
	for _, c := range comments[1:] {
		assert.True(t, LooksLikeRecipeComment(c.Message), "非置顶评论必须像菜谱: %s", c.Message)
	}
}

func TestGetRecipeLikeComments_RespectsLimit(t *testing.T) {
	server := newFakeServer(t)
	client := newTestClient(server)

	comments, err := client.GetRecipeLikeComments(context.Background(), 123456, 2)
	require.NoError(t, err)
	assert.Len(t, comments, 2)
}

func TestLooksLikeRecipeComment(t *testing.T) {
	assert.True(t, LooksLikeRecipeComment("五花肉500克，焯水后小火炖"))
	assert.False(t, LooksLikeRecipeComment("看起来好好吃"))
	assert.False(t, LooksLikeRecipeComment("只放了一点盐"))
}
