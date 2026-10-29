package com.wanli.android.presentation.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.theme.CardWhite
import com.wanli.android.presentation.theme.WarmShadow

/** 设计系统标准卡片：纯白、24dp 圆角、暖色柔阴影。 */
@Composable
fun WanLiCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val shape = MaterialTheme.shapes.large
    Column(
        modifier = modifier
            .shadow(elevation = 8.dp, shape = shape, ambientColor = WarmShadow, spotColor = WarmShadow)
            .clip(shape)
            .background(CardWhite)
            .then(if (onClick != null) Modifier.clickable(onClick = onClick) else Modifier),
        content = content,
    )
}

/** 卡片内容统一内边距修饰符。 */
fun Modifier.cardContentPadding(): Modifier = this.padding(16.dp)
