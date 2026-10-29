package parser

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseBVID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "标准视频链接",
			input: "https://www.bilibili.com/video/BV1xx411c7xx/",
			want:  "BV1xx411c7xx",
		},
		{
			name:  "带参数的链接",
			input: "https://www.bilibili.com/video/BV1xx411c7xx/?spm_id_from=333.337&vd_source=abc",
			want:  "BV1xx411c7xx",
		},
		{
			name:  "纯 BV 号",
			input: "BV1xx411c7xx",
			want:  "BV1xx411c7xx",
		},
		{
			name:  "前后带空白",
			input: "  https://www.bilibili.com/video/BV1xx411c7xx/ \n",
			want:  "BV1xx411c7xx",
		},
		{
			name:  "分享文案中夹带链接",
			input: "【这个红烧肉太香了-哔哩哔哩】 https://www.bilibili.com/video/BV1xx411c7xx",
			want:  "BV1xx411c7xx",
		},
		{
			name:    "空输入",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "非 B 站链接",
			input:   "https://www.youtube.com/watch?v=abc",
			wantErr: true,
		},
		{
			name:    "BV 号长度不足",
			input:   "https://www.bilibili.com/video/BV1xx",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBVID(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestExtractURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "B站分享文案",
			input: "【隋卞一做 |怎么又简单又好吃啊！有手就行版本的番茄炖牛腩！-哔哩哔哩】 https://b23.tv/osM7T47",
			want:  "https://b23.tv/osM7T47",
		},
		{
			name:  "链接后紧跟中文标点",
			input: "【标题】https://b23.tv/osM7T47，复制链接打开",
			want:  "https://b23.tv/osM7T47",
		},
		{
			name:  "不带协议头的短链",
			input: "【标题-哔哩哔哩】 b23.tv/osM7T47",
			want:  "b23.tv/osM7T47",
		},
		{
			name:  "纯链接保持不变",
			input: "  https://www.bilibili.com/video/BV1xx411c7xx/?p=1  ",
			want:  "https://www.bilibili.com/video/BV1xx411c7xx/?p=1",
		},
		{
			name:  "没有链接时返回原文",
			input: "  随便一段文字  ",
			want:  "随便一段文字",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ExtractURL(tt.input))
		})
	}
}

func TestParseBVID_ShareText(t *testing.T) {
	// 分享文案里的完整链接应能直接提取 BV 号
	got, err := ParseBVID("【这个红烧肉太香了-哔哩哔哩】 https://www.bilibili.com/video/BV1xx411c7xx")
	require.NoError(t, err)
	assert.Equal(t, "BV1xx411c7xx", got)
}

func TestResolveShortURL(t *testing.T) {
	const realURL = "https://www.bilibili.com/video/BV1xx411c7xx/"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, realURL, http.StatusFound)
	}))
	defer server.Close()

	got, err := ResolveShortURL(server.URL)
	require.NoError(t, err)
	assert.Equal(t, realURL, got)
}

func TestResolveShortURL_NoRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	_, err := ResolveShortURL(server.URL)
	assert.Error(t, err)
}

func TestIsValidBilibiliURL(t *testing.T) {
	valid := []string{
		"https://www.bilibili.com/video/BV1xx411c7xx/",
		"https://b23.tv/abc123",
		"BV1xx411c7xx",
		"bili.im/xyz",
	}
	for _, input := range valid {
		assert.True(t, IsValidBilibiliURL(input), "应识别为有效: %s", input)
	}

	invalid := []string{"", "https://www.youtube.com/watch?v=abc", "随便一段文字"}
	for _, input := range invalid {
		assert.False(t, IsValidBilibiliURL(input), "应识别为无效: %s", input)
	}
}
