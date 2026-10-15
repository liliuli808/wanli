package com.wanli.android.presentation.components

import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.theme.CardWhite

/** 设计系统标准卡片：纯白、16dp 圆角、柔和投影。 */
@Composable
fun WanLiCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val colors = CardDefaults.cardColors(containerColor = CardWhite)
    val elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    val shape = MaterialTheme.shapes.medium
    if (onClick != null) {
        Card(onClick = onClick, modifier = modifier, shape = shape, colors = colors, elevation = elevation, content = content)
    } else {
        Card(modifier = modifier, shape = shape, colors = colors, elevation = elevation, content = content)
    }
}

/** 卡片内容统一内边距修饰符。 */
fun Modifier.cardContentPadding(): Modifier = this.padding(16.dp)
