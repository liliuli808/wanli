package com.wanli.android.presentation.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.outlined.StarOutline
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.wanli.android.presentation.theme.MustardAccent
import com.wanli.android.presentation.theme.MustardOnAccent
import com.wanli.android.presentation.theme.SageSoft

/** 圆形用户头像：有图显示图，无图显示昵称首字（鼠尾草绿底）。 */
@Composable
fun UserAvatar(
    nickname: String,
    avatarUrl: String?,
    size: Dp = 36.dp,
    modifier: Modifier = Modifier,
) {
    if (!avatarUrl.isNullOrBlank()) {
        AsyncImage(
            model = avatarUrl,
            contentDescription = nickname,
            modifier = modifier.size(size).clip(CircleShape),
            contentScale = ContentScale.Crop,
        )
    } else {
        Box(
            modifier = modifier.size(size).clip(CircleShape).background(SageSoft),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                text = nickname.firstOrNull()?.toString() ?: "碗",
                style = MaterialTheme.typography.titleSmall,
                color = MaterialTheme.colorScheme.primary,
            )
        }
    }
}

/** 作者行：头像 + 昵称 + 可选副信息（时间等）。 */
@Composable
fun AuthorRow(
    nickname: String,
    modifier: Modifier = Modifier,
    avatarUrl: String? = null,
    subtitle: String? = null,
    trailing: (@Composable RowScope.() -> Unit)? = null,
) {
    Row(modifier = modifier, verticalAlignment = Alignment.CenterVertically) {
        UserAvatar(nickname = nickname, avatarUrl = avatarUrl, size = 32.dp)
        Spacer(Modifier.width(8.dp))
        Column(Modifier.weight(1f)) {
            Text(
                text = nickname,
                style = MaterialTheme.typography.titleSmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            if (subtitle != null) {
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        trailing?.invoke(this)
    }
}

/** 评分星（芥末黄）。rating 为 null 时不渲染。 */
@Composable
fun RatingStars(rating: Int?, modifier: Modifier = Modifier, starSize: Dp = 16.dp) {
    if (rating == null) return
    val safe = rating.coerceIn(0, 5)
    Row(
        modifier = modifier.semantics(mergeDescendants = true) {
            contentDescription = "$safe 星"
        },
    ) {
        repeat(5) { index ->
            Icon(
                imageVector = if (index < safe) Icons.Filled.Star else Icons.Outlined.StarOutline,
                contentDescription = null,
                tint = MustardAccent,
                modifier = Modifier.size(starSize),
            )
        }
    }
}

/** 芥末黄社交徽章，如「12 人做过」「AI 推测」。 */
@Composable
fun SocialBadge(text: String, modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier,
        shape = MaterialTheme.shapes.small,
        color = MustardAccent,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Bold,
            color = MustardOnAccent,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp),
        )
    }
}
