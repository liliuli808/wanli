package com.wanli.android.presentation.main

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ContentPaste
import androidx.compose.material.icons.outlined.Info
import androidx.compose.material.icons.outlined.PhotoLibrary
import androidx.compose.material.icons.outlined.SmartDisplay
import androidx.compose.material3.Button
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.OutlinedTextFieldDefaults
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.home.ImportUiState
import com.wanli.android.presentation.theme.CaramelAccent
import com.wanli.android.presentation.theme.InkSecondary
import com.wanli.android.presentation.theme.OnTomato
import com.wanli.android.presentation.theme.OutlineWarm
import com.wanli.android.presentation.theme.SurfaceWarm
import com.wanli.android.presentation.theme.TomatoRed
import com.wanli.android.presentation.theme.TomatoSoft

/** 中央 FAB 打开的「导入新菜谱」底部弹层内容。 */
@Composable
fun ImportSheetContent(
    url: String,
    onUrlChange: (String) -> Unit,
    importState: ImportUiState,
    onSubmit: () -> Unit,
    onPaste: () -> Unit,
    onOpenRecipe: (String) -> Unit,
    onProofread: (String) -> Unit,
    onNavigateToLogin: () -> Unit,
    onReset: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val busy = importState is ImportUiState.Submitting || importState is ImportUiState.Polling
    Column(modifier = modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 8.dp)) {
        SheetHandle()
        Spacer(Modifier.height(16.dp))
        Text("导入新菜谱", style = MaterialTheme.typography.headlineSmall)
        Spacer(Modifier.height(6.dp))
        Text(
            "粘贴 B 站视频链接，AI 帮你拆成步骤",
            style = MaterialTheme.typography.bodyMedium,
            color = InkSecondary,
        )
        Spacer(Modifier.height(20.dp))
        OutlinedTextField(
            value = url,
            onValueChange = onUrlChange,
            modifier = Modifier.fillMaxWidth(),
            singleLine = true,
            enabled = !busy,
            placeholder = { Text("https://www.bilibili.com/video/…") },
            leadingIcon = { Icon(Icons.Outlined.SmartDisplay, contentDescription = null) },
            trailingIcon = {
                TextButton(onClick = onPaste, enabled = !busy) { Text("粘贴") }
            },
            shape = MaterialTheme.shapes.medium,
            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Go),
            keyboardActions = KeyboardActions(onGo = { if (url.isNotBlank()) onSubmit() }),
            colors = OutlinedTextFieldDefaults.colors(
                focusedContainerColor = SurfaceWarm,
                unfocusedContainerColor = SurfaceWarm,
                disabledContainerColor = SurfaceWarm,
                focusedBorderColor = TomatoRed,
                unfocusedBorderColor = OutlineWarm,
                disabledBorderColor = OutlineWarm,
            ),
        )
        Spacer(Modifier.height(10.dp))
        Row(verticalAlignment = Alignment.CenterVertically) {
            Icon(
                Icons.Outlined.Info,
                contentDescription = null,
                tint = CaramelAccent,
                modifier = Modifier.size(16.dp),
            )
            Spacer(Modifier.width(6.dp))
            Text(
                "需视频带字幕，解析约 20 秒",
                style = MaterialTheme.typography.bodySmall,
                color = InkSecondary,
            )
        }
        Spacer(Modifier.height(20.dp))
        Button(
            onClick = onSubmit,
            enabled = !busy && url.isNotBlank(),
            modifier = Modifier.fillMaxWidth().height(52.dp),
            shape = CircleShape,
            colors = ButtonDefaults.buttonColors(containerColor = TomatoRed, contentColor = OnTomato),
        ) {
            if (busy) {
                CircularProgressIndicator(
                    strokeWidth = 2.dp,
                    color = OnTomato,
                    modifier = Modifier.size(18.dp),
                )
                Spacer(Modifier.width(10.dp))
            }
            Text(
                when (importState) {
                    is ImportUiState.Submitting -> "正在提交…"
                    is ImportUiState.Polling -> importState.message
                    else -> "开始解析"
                },
                style = MaterialTheme.typography.labelLarge,
                fontWeight = FontWeight.Bold,
            )
        }
        Spacer(Modifier.height(10.dp))
        OutlinedButton(
            onClick = onPaste,
            enabled = !busy,
            modifier = Modifier.fillMaxWidth().height(52.dp),
            shape = CircleShape,
        ) {
            Icon(Icons.Outlined.PhotoLibrary, contentDescription = null, modifier = Modifier.size(18.dp))
            Spacer(Modifier.width(8.dp))
            Text("从相册导入图片")
        }
        ImportResultBlock(
            importState = importState,
            onOpenRecipe = onOpenRecipe,
            onProofread = onProofread,
            onNavigateToLogin = onNavigateToLogin,
            onReset = onReset,
        )
        Spacer(Modifier.height(24.dp))
    }
}

@Composable
private fun SheetHandle() {
    Box(
        modifier = Modifier
            .padding(top = 8.dp)
            .width(40.dp)
            .height(4.dp)
            .clip(CircleShape)
            .background(OutlineWarm),
    )
}

@Composable
private fun ImportResultBlock(
    importState: ImportUiState,
    onOpenRecipe: (String) -> Unit,
    onProofread: (String) -> Unit,
    onNavigateToLogin: () -> Unit,
    onReset: () -> Unit,
) {
    val (title, tint) = when (importState) {
        is ImportUiState.Completed -> "解析完成，已生成待校对草稿" to TomatoRed
        is ImportUiState.Failed -> importState.message to MaterialTheme.colorScheme.error
        is ImportUiState.Duplicate -> "该视频已有解析菜谱" to CaramelAccent
        is ImportUiState.LoginRequired -> "登录后才能解析视频" to CaramelAccent
        else -> return
    }
    Spacer(Modifier.height(16.dp))
    Column(
        Modifier
            .fillMaxWidth()
            .clip(MaterialTheme.shapes.medium)
            .background(TomatoSoft)
            .padding(16.dp),
    ) {
        Text(title, style = MaterialTheme.typography.bodyMedium, color = tint)
        Spacer(Modifier.height(10.dp))
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            when (importState) {
                is ImportUiState.Completed -> {
                    importState.recipeVersionId?.let { id ->
                        Button(onClick = { onProofread(id) }, shape = CircleShape) { Text("去校对发布") }
                    }
                    TextButton(onClick = onReset) { Text("继续解析") }
                }
                is ImportUiState.Duplicate -> {
                    importState.recipeVersionId?.let { id ->
                        Button(onClick = { onOpenRecipe(id) }, shape = CircleShape) { Text("查看菜谱") }
                    }
                    TextButton(onClick = onReset) { Text("知道了") }
                }
                is ImportUiState.LoginRequired -> {
                    Button(onClick = onNavigateToLogin, shape = CircleShape) { Text("去登录") }
                }
                is ImportUiState.Failed -> {
                    Button(onClick = onReset, shape = CircleShape) { Text("重试") }
                }
                else -> Unit
            }
        }
    }
}
