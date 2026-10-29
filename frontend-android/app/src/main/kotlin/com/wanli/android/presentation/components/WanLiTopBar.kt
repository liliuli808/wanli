package com.wanli.android.presentation.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.theme.CardWhite
import com.wanli.android.presentation.theme.WarmShadow

/**
 * 「碗里」统一顶栏。
 * @param onBack 传入时显示返回按钮；为 null 时作为一级页面标题栏。
 * @param actions 右侧操作区（如铃铛按钮）。
 */
@Composable
fun WanLiTopBar(
    title: String,
    onBack: (() -> Unit)? = null,
    actions: (@Composable () -> Unit)? = null,
) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 20.dp, vertical = 12.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        if (onBack != null) {
            IconButton(
                onClick = onBack,
                modifier = Modifier
                    .size(40.dp)
                    .shadow(8.dp, CircleShape, ambientColor = WarmShadow, spotColor = WarmShadow)
                    .clip(CircleShape)
                    .background(CardWhite),
            ) {
                Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "返回")
            }
        }
        Text(
            text = title,
            style = MaterialTheme.typography.titleLarge,
            modifier = Modifier
                .weight(1f)
                .padding(start = if (onBack != null) 12.dp else 0.dp),
        )
        actions?.invoke()
    }
}

/** 顶栏右侧的白色圆形图标按钮（铃铛 / 搜索 / 设置）。 */
@Composable
fun CircleIconButton(
    onClick: () -> Unit,
    contentDescription: String?,
    modifier: Modifier = Modifier,
    content: @Composable () -> Unit,
) {
    IconButton(
        onClick = onClick,
        modifier = modifier
            .size(40.dp)
            .shadow(8.dp, CircleShape, ambientColor = WarmShadow, spotColor = WarmShadow)
            .clip(CircleShape)
            .background(CardWhite),
    ) {
        content()
    }
}
