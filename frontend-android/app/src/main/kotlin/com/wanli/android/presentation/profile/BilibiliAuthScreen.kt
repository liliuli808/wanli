package com.wanli.android.presentation.profile

import android.annotation.SuppressLint
import android.view.ViewGroup
import android.webkit.CookieManager
import android.webkit.WebView
import android.webkit.WebViewClient
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.unit.dp
import androidx.compose.ui.viewinterop.AndroidView
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ErrorOutline
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.StatusBanner
import com.wanli.android.presentation.components.StatusBannerType
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding
import kotlinx.coroutines.delay

private const val LOGIN_URL = "https://passport.bilibili.com/login"
private const val COOKIE_URL = "https://www.bilibili.com"

// B 站会拦截 WebView 默认 UA，这里伪装成手机 Chrome，避免登录页直接拒绝加载。
private const val MOBILE_CHROME_UA =
    "Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 " +
        "(KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36"

private val sessdataRegex = Regex("(?:^|;\\s*)SESSDATA=([^;]+)")

@Composable
fun BilibiliAuthScreen(
    onBack: () -> Unit,
    onAuthorized: () -> Unit = onBack,
    viewModel: BilibiliAuthViewModel = hiltViewModel(),
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    var showManualInput by rememberSaveable { mutableStateOf(false) }

    LaunchedEffect(state.bindSucceeded) {
        if (state.bindSucceeded) onAuthorized()
    }

    if (showManualInput) {
        ManualSessdataDialog(
            onDismiss = { showManualInput = false },
            onConfirm = { sessdata ->
                showManualInput = false
                viewModel.bind(sessdata)
            },
        )
    }

    Scaffold(
        topBar = { WanLiTopBar(title = "B 站账号", onBack = onBack) },
    ) { padding ->
        Column(modifier = Modifier.padding(padding).fillMaxSize()) {
            when {
                state.loading -> Column(
                    modifier = Modifier.fillMaxSize(),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.Center,
                ) { CircularProgressIndicator() }

                state.loadError != null -> EmptyState(
                    icon = Icons.Outlined.ErrorOutline,
                    title = "加载失败",
                    subtitle = state.loadError ?: "",
                    primaryLabel = "重试",
                    onPrimary = viewModel::load,
                    modifier = Modifier.fillMaxSize(),
                )

                state.bound -> BoundContent(
                    uname = state.uname,
                    mid = state.mid,
                    busy = state.binding,
                    message = state.message,
                    onUnbind = viewModel::unbind,
                    onDismissMessage = viewModel::clearMessage,
                )

                else -> UnboundContent(
                    binding = state.binding,
                    message = state.message,
                    onDetectSessdata = viewModel::bind,
                    onManualInput = { showManualInput = true },
                    onDismissMessage = viewModel::clearMessage,
                )
            }
        }
    }
}

@Composable
private fun UnboundContent(
    binding: Boolean,
    message: String?,
    onDetectSessdata: (String) -> Unit,
    onManualInput: () -> Unit,
    onDismissMessage: () -> Unit,
) {
    Column(modifier = Modifier.fillMaxSize()) {
        Text(
            text = "部分视频的字幕只有登录后才能读取。授权后，碗里会用你的 B 站账号读取字幕，仅用于菜谱解析。",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 20.dp, vertical = 12.dp),
        )

        if (message != null) {
            StatusBanner(
                type = StatusBannerType.ERROR,
                modifier = Modifier.padding(horizontal = 20.dp, vertical = 8.dp),
            ) {
                Text(message, style = MaterialTheme.typography.bodyMedium)
                TextButton(onClick = onDismissMessage) { Text("知道了") }
            }
        }

        if (binding) {
            Row(
                modifier = Modifier.fillMaxWidth().padding(16.dp),
                horizontalArrangement = Arrangement.Center,
                verticalAlignment = Alignment.CenterVertically,
            ) {
                CircularProgressIndicator(modifier = Modifier.height(20.dp))
                Spacer(Modifier.width(8.dp))
                Text("正在校验授权…", style = MaterialTheme.typography.bodyMedium)
            }
        }

        BilibiliLoginWebView(
            onSessdata = onDetectSessdata,
            modifier = Modifier.weight(1f).fillMaxWidth(),
        )

        TextButton(
            onClick = onManualInput,
            modifier = Modifier.align(Alignment.CenterHorizontally).padding(bottom = 12.dp),
        ) {
            Text("登录页加载不出来？手动粘贴 SESSDATA")
        }
    }
}

@Composable
private fun BoundContent(
    uname: String,
    mid: String,
    busy: Boolean,
    message: String?,
    onUnbind: () -> Unit,
    onDismissMessage: () -> Unit,
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(20.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        WanLiCard(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.cardContentPadding()) {
                Text("已授权 B 站账号", style = MaterialTheme.typography.titleMedium)
                Spacer(Modifier.height(8.dp))
                Text(
                    text = uname.ifBlank { "B 站用户" },
                    style = MaterialTheme.typography.headlineSmall,
                )
                if (mid.isNotBlank()) {
                    Text(
                        text = "UID $mid",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        Text(
            text = "凭证已加密保存在服务端，仅用于读取字幕；碗里不会用它做其他操作。",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
        )

        if (message != null) {
            StatusBanner(type = StatusBannerType.INFO) {
                Text(message, style = MaterialTheme.typography.bodyMedium)
                TextButton(onClick = onDismissMessage) { Text("知道了") }
            }
        }

        OutlinedButton(
            onClick = onUnbind,
            enabled = !busy,
            modifier = Modifier.fillMaxWidth(),
        ) {
            Text(if (busy) "处理中…" else "解除授权")
        }
    }
}

/**
 * 内嵌 B 站登录页。登录成功后 CookieManager 会拿到 SESSDATA，
 * 这里轮询监听并回调，不需要解析页面 DOM。
 */
@SuppressLint("SetJavaScriptEnabled")
@Composable
private fun BilibiliLoginWebView(
    onSessdata: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    val cookieManager = remember {
        CookieManager.getInstance().apply { setAcceptCookie(true) }
    }
    var webView by remember { mutableStateOf<WebView?>(null) }

    DisposableEffect(Unit) {
        onDispose {
            webView?.let { view ->
                (view.parent as? ViewGroup)?.removeView(view)
                view.destroy()
            }
            webView = null
        }
    }

    AndroidView(
        modifier = modifier,
        factory = { context ->
            WebView(context).apply {
                layoutParams = ViewGroup.LayoutParams(
                    ViewGroup.LayoutParams.MATCH_PARENT,
                    ViewGroup.LayoutParams.MATCH_PARENT,
                )
                settings.apply {
                    javaScriptEnabled = true
                    domStorageEnabled = true
                    databaseEnabled = true
                    userAgentString = MOBILE_CHROME_UA
                }
                // B 站登录依赖第三方 Cookie（passport → www），必须显式开启
                cookieManager.setAcceptThirdPartyCookies(this, true)
                webViewClient = WebViewClient()
                loadUrl(LOGIN_URL)
                webView = this
            }
        },
    )

    LaunchedEffect(Unit) {
        // 持续监听：首次拿到 SESSDATA 就提交；同一值只提交一次，
        // 用户换账号（值变化）时会自动重新提交。
        var lastSubmitted: String? = null
        while (true) {
            val cookie = cookieManager.getCookie(COOKIE_URL)
            val sessdata = cookie?.let { sessdataRegex.find(it)?.groupValues?.get(1) }
            if (!sessdata.isNullOrBlank() && sessdata != lastSubmitted) {
                lastSubmitted = sessdata
                onSessdata(sessdata)
            }
            delay(1000)
        }
    }
}

@Composable
private fun ManualSessdataDialog(
    onDismiss: () -> Unit,
    onConfirm: (String) -> Unit,
) {
    var input by rememberSaveable { mutableStateOf("") }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("手动粘贴 SESSDATA") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(
                    "在电脑浏览器登录 bilibili.com，打开开发者工具 → Application → Cookies，" +
                        "复制 SESSDATA 的值粘贴到这里。",
                    style = MaterialTheme.typography.bodySmall,
                )
                OutlinedTextField(
                    value = input,
                    onValueChange = { input = it },
                    label = { Text("SESSDATA") },
                    singleLine = true,
                    visualTransformation = PasswordVisualTransformation(),
                    modifier = Modifier.fillMaxWidth(),
                )
            }
        },
        confirmButton = {
            Button(
                onClick = { onConfirm(input) },
                enabled = input.isNotBlank(),
            ) { Text("授权") }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("取消") }
        },
    )
}
