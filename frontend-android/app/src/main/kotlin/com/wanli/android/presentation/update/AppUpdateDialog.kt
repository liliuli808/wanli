package com.wanli.android.presentation.update

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.LinearProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.wanli.android.BuildConfig
import java.util.Locale

/**
 * 全局更新弹窗。挂在 MainActivity 的 Compose 根节点上，
 * 因此无论用户停在哪个页面都能弹出。
 */
@Composable
fun AppUpdateDialog(
    state: AppUpdateUiState,
    onDismiss: () -> Unit,
    onDownload: () -> Unit,
    onInstall: () -> Unit,
) {
    when (state) {
        is AppUpdateUiState.Idle -> Unit

        is AppUpdateUiState.Checking -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text("检查更新") },
            text = { Text("正在检查是否有新版本…") },
            confirmButton = {},
            dismissButton = { TextButton(onClick = onDismiss) { Text("取消") } },
        )

        is AppUpdateUiState.UpToDate -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text("已是最新版本") },
            text = { Text("当前版本 ${state.currentVersion}，先用着吧。") },
            confirmButton = { TextButton(onClick = onDismiss) { Text("好") } },
        )

        is AppUpdateUiState.Available -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text("发现新版本 ${state.info.versionName}") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    Text(
                        "当前版本 ${BuildConfig.VERSION_NAME}" +
                            if (state.info.size > 0) " · ${formatSize(state.info.size)}" else "",
                        style = MaterialTheme.typography.bodyMedium,
                    )
                    Text(
                        "更新会下载安装包并覆盖安装，菜谱与账号数据不会丢失。",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            },
            confirmButton = { TextButton(onClick = onDownload) { Text("立即更新") } },
            dismissButton = { TextButton(onClick = onDismiss) { Text("稍后") } },
        )

        is AppUpdateUiState.Downloading -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text("正在下载更新") },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(12.dp)) {
                    LinearProgressIndicator(
                        progress = { state.progress },
                        modifier = Modifier.fillMaxWidth(),
                    )
                    Text(
                        text = if (state.total > 0) {
                            "${(state.progress * 100).toInt()}%  " +
                                "${formatSize(state.downloaded)} / ${formatSize(state.total)}"
                        } else {
                            "已下载 ${formatSize(state.downloaded)}"
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            },
            confirmButton = {},
            dismissButton = { TextButton(onClick = onDismiss) { Text("取消") } },
        )

        is AppUpdateUiState.Ready -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text("下载完成") },
            text = {
                Text("新版本 ${state.info.versionName} 已准备好，点击「安装」完成更新。")
            },
            confirmButton = { TextButton(onClick = onInstall) { Text("安装") } },
            dismissButton = { TextButton(onClick = onDismiss) { Text("稍后") } },
        )

        is AppUpdateUiState.Failed -> AlertDialog(
            onDismissRequest = onDismiss,
            title = { Text("更新失败") },
            text = { Text(state.message) },
            confirmButton = {
                if (state.info != null) {
                    TextButton(onClick = onDownload) { Text("重试") }
                }
            },
            dismissButton = { TextButton(onClick = onDismiss) { Text("关闭") } },
        )
    }
}

private fun formatSize(bytes: Long): String {
    if (bytes <= 0) return "0 B"
    val kb = bytes / 1024.0
    if (kb < 1024) return String.format(Locale.US, "%.0f KB", kb)
    return String.format(Locale.US, "%.1f MB", kb / 1024)
}
