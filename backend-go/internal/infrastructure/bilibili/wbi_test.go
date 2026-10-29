package bilibili

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testImgKey = "7cd084941338484aae1ad9425b84077c"
	testSubKey = "4932caff0ff746eab6f01bf08b70ac45"
)

func TestMixinKey(t *testing.T) {
	// 与 B 站文档示例一致：img/sub 组合后取前 32 位
	assert.Equal(t, "ea1db124af3c7062474693fa704f4ff8", mixinKey(testImgKey, testSubKey))
}

func TestSign(t *testing.T) {
	params := url.Values{
		"bvid": {"BV1RN4y1r7dt"},
		"cid":  {"1309825896"},
	}
	now := time.Unix(1700000000, 0)

	query := sign(params, mixinKey(testImgKey, testSubKey), now)
	assert.Equal(t,
		"bvid=BV1RN4y1r7dt&cid=1309825896&wts=1700000000&w_rid=23b59bcb0e4f6973d1a081901eae4704",
		query)
}

func TestSign_FiltersSpecialChars(t *testing.T) {
	params := url.Values{"keyword": {"a!b'c(d)e*f"}}
	query := sign(params, "key", time.Unix(1700000000, 0))
	assert.Contains(t, query, "keyword=abcdef")
}

func TestKeyFilename(t *testing.T) {
	assert.Equal(t, "7cd084941338484aae1ad9425b84077c",
		keyFilename("https://i0.hdslb.com/bfs/wbi/7cd084941338484aae1ad9425b84077c.png"))
	assert.Equal(t, "", keyFilename(""))
}

func TestFetchPlayerSubtitles_SignsRequestAndCachesKeys(t *testing.T) {
	var navCalls atomic.Int32
	var lastQuery string

	mux := http.NewServeMux()
	mux.HandleFunc("/x/web-interface/nav", func(w http.ResponseWriter, r *http.Request) {
		navCalls.Add(1)
		fmt.Fprintf(w, `{"code":-101,"data":{"wbi_img":{
			"img_url":"https://i0.hdslb.com/bfs/wbi/%s.png",
			"sub_url":"https://i0.hdslb.com/bfs/wbi/%s.png"}}}`, testImgKey, testSubKey)
	})
	mux.HandleFunc("/x/player/wbi/v2", func(w http.ResponseWriter, r *http.Request) {
		lastQuery = r.URL.RawQuery
		fmt.Fprint(w, `{"code":0,"data":{"need_login_subtitle":false,"subtitle":{"subtitles":[]}}}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := New(server.URL, nil)
	info := &VideoInfo{BVID: "BV1RN4y1r7dt", CID: 1309825896}

	_, err := client.fetchPlayerSubtitles(context.Background(), info)
	require.NoError(t, err)
	_, err = client.fetchPlayerSubtitles(context.Background(), info)
	require.NoError(t, err)

	assert.Contains(t, lastQuery, "w_rid=")
	assert.Contains(t, lastQuery, "wts=")
	assert.Equal(t, int32(1), navCalls.Load(), "wbi 密钥应被缓存复用")
}

func TestFetchPlayerSubtitles_RefreshesKeysOn403(t *testing.T) {
	var navCalls atomic.Int32
	var playerCalls atomic.Int32

	mux := http.NewServeMux()
	mux.HandleFunc("/x/web-interface/nav", func(w http.ResponseWriter, r *http.Request) {
		navCalls.Add(1)
		fmt.Fprintf(w, `{"code":-101,"data":{"wbi_img":{
			"img_url":"https://i0.hdslb.com/bfs/wbi/%s.png",
			"sub_url":"https://i0.hdslb.com/bfs/wbi/%s.png"}}}`, testImgKey, testSubKey)
	})
	mux.HandleFunc("/x/player/wbi/v2", func(w http.ResponseWriter, r *http.Request) {
		if playerCalls.Add(1) == 1 {
			fmt.Fprint(w, `{"code":-403,"message":"访问权限不足"}`)
			return
		}
		fmt.Fprint(w, `{"code":0,"data":{"subtitle":{"subtitles":[]}}}`)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := New(server.URL, nil)
	info := &VideoInfo{BVID: "BV1RN4y1r7dt", CID: 1309825896}

	_, err := client.fetchPlayerSubtitles(context.Background(), info)
	require.NoError(t, err)
	assert.Equal(t, int32(2), navCalls.Load(), "403 后应重新拉取 wbi 密钥")
	assert.Equal(t, int32(2), playerCalls.Load())
}

func TestWithCookie_SharesWBICache(t *testing.T) {
	base := New("http://unused", nil)
	auth := base.WithCookie("SESSDATA=abc")

	require.NotSame(t, base, auth)
	assert.Equal(t, "SESSDATA=abc", auth.cookie)
	assert.Equal(t, "", base.cookie, "原客户端不应被修改")
	assert.Same(t, base.wbi, auth.wbi, "副本应共享 wbi 密钥缓存")
	assert.Same(t, base, base.WithCookie(""), "空 Cookie 应返回原客户端")
}

func TestNav(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    *NavInfo
		wantErr bool
	}{
		{
			name: "已登录",
			body: `{"code":0,"data":{"isLogin":true,"mid":123456,"uname":"测试用户"}}`,
			want: &NavInfo{IsLogin: true, Mid: 123456, Uname: "测试用户"},
		},
		{
			name: "未登录",
			body: `{"code":-101,"message":"账号未登录","data":{"isLogin":false}}`,
			want: &NavInfo{IsLogin: false},
		},
		{
			name:    "接口错误",
			body:    `{"code":-500,"message":"服务器错误"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.True(t, strings.HasSuffix(r.URL.Path, "/x/web-interface/nav"))
				fmt.Fprint(w, tt.body)
			}))
			t.Cleanup(server.Close)

			got, err := New(server.URL, nil).Nav(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
