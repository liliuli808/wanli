package com.wanli.android.presentation.auth

import android.annotation.SuppressLint
import android.webkit.JavascriptInterface
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.viewinterop.AndroidView

/**
 * 用 WebView 加载后端托管的 Turnstile 组件页，并把 token/错误回调给 Compose 层。
 * Cloudflare 官方对原生 App 的标准接入方式即 WebView（无原生 SDK）。
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
fun TurnstileWidget(
    widgetUrl: String,
    onTokenReceived: (String) -> Unit,
    onError: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    AndroidView(
        modifier = modifier,
        factory = { context ->
            WebView(context).apply {
                settings.javaScriptEnabled = true
                settings.domStorageEnabled = true
                addJavascriptInterface(
                    object {
                        @JavascriptInterface
                        fun onTokenReceived(token: String) = onTokenReceived(token)

                        @JavascriptInterface
                        fun onError(code: String) = onError(code)
                    },
                    "AndroidBridge"
                )
                webViewClient = WebViewClient()
                loadUrl(widgetUrl)
            }
        }
    )
}
