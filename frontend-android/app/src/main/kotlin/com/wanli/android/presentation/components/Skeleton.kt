package com.wanli.android.presentation.components

import androidx.compose.animation.core.RepeatMode
import androidx.compose.animation.core.animateFloat
import androidx.compose.animation.core.infiniteRepeatable
import androidx.compose.animation.core.rememberInfiniteTransition
import androidx.compose.animation.core.tween
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.theme.CreamSurfaceVariant
import com.wanli.android.presentation.theme.CardWhite

/** 呼吸骨架卡片：列表加载态占位。 */
@Composable
fun SkeletonCard(modifier: Modifier = Modifier) {
    val transition = rememberInfiniteTransition(label = "skeleton")
    val alpha by transition.animateFloat(
        initialValue = 0.4f, targetValue = 1f,
        animationSpec = infiniteRepeatable(animation = tween(800), repeatMode = RepeatMode.Reverse),
        label = "alpha",
    )
    Column(modifier.fillMaxWidth().clip(MaterialTheme.shapes.medium).background(CardWhite).padding(16.dp)) {
        Box(Modifier.fillMaxWidth().height(160.dp).clip(MaterialTheme.shapes.small).background(CreamSurfaceVariant.copy(alpha = alpha)))
        Spacer(Modifier.height(12.dp))
        Box(Modifier.fillMaxWidth(0.6f).height(16.dp).clip(MaterialTheme.shapes.small).background(CreamSurfaceVariant.copy(alpha = alpha)))
        Spacer(Modifier.height(8.dp))
        Box(Modifier.fillMaxWidth(0.4f).height(14.dp).clip(MaterialTheme.shapes.small).background(CreamSurfaceVariant.copy(alpha = alpha)))
    }
}
