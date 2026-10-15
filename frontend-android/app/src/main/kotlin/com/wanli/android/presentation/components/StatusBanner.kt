package com.wanli.android.presentation.components

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CheckCircle
import androidx.compose.material.icons.filled.Info
import androidx.compose.material.icons.filled.Warning
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.theme.ErrorRed40
import com.wanli.android.presentation.theme.ErrorRed90
import com.wanli.android.presentation.theme.InfoSage40
import com.wanli.android.presentation.theme.InfoSage90
import com.wanli.android.presentation.theme.SuccessGreen40
import com.wanli.android.presentation.theme.SuccessGreen90
import com.wanli.android.presentation.theme.WarningAmber40
import com.wanli.android.presentation.theme.WarningAmber90

/** 状态条语义类型，决定图标与配色。 */
enum class StatusBannerType(
    val icon: ImageVector,
    val containerColor: Color,
    val contentColor: Color
) {
    INFO(
        icon = Icons.Default.Info,
        containerColor = InfoSage90,
        contentColor = InfoSage40
    ),
    SUCCESS(
        icon = Icons.Default.CheckCircle,
        containerColor = SuccessGreen90,
        contentColor = SuccessGreen40
    ),
    WARNING(
        icon = Icons.Default.Warning,
        containerColor = WarningAmber90,
        contentColor = WarningAmber40
    ),
    ERROR(
        icon = Icons.Default.Warning,
        containerColor = ErrorRed90,
        contentColor = ErrorRed40
    )
}

/**
 * 带图标与语义配色的状态条卡片。
 * 用于解析进度、完成、失败、重复视频等即时反馈场景。
 */
@Composable
fun StatusBanner(
    type: StatusBannerType,
    modifier: Modifier = Modifier,
    content: @Composable ColumnScope.() -> Unit
) {
    Card(
        colors = CardDefaults.cardColors(
            containerColor = type.containerColor,
            contentColor = type.contentColor
        ),
        modifier = modifier.fillMaxWidth()
    ) {
        Row(
            verticalAlignment = Alignment.Top,
            modifier = Modifier.padding(16.dp)
        ) {
            Icon(
                imageVector = type.icon,
                contentDescription = null,
                tint = type.contentColor,
                modifier = Modifier.size(22.dp)
            )
            Spacer(modifier = Modifier.width(12.dp))
            Column(content = content)
        }
    }
}
