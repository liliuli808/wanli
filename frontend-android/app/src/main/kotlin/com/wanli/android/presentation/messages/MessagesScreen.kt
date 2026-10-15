package com.wanli.android.presentation.messages

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ChatBubbleOutline
import androidx.compose.material.icons.outlined.CheckCircle
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.RateReview
import androidx.compose.material.icons.outlined.Restaurant
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.data.remote.MessageDto
import com.wanli.android.presentation.components.AuthorRow
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.RatingStars
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.ScrollableFill
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding
import com.wanli.android.presentation.theme.SuccessGreen40
import com.wanli.android.presentation.theme.WarningAmber40

private const val TYPE_COMMENT = "comment"
private const val TYPE_COOKING_RECORD = "cooking_record"
private const val TYPE_CORRECTION_REVIEWED = "correction_reviewed"
private const val TYPE_CORRECTION_PENDING = "correction_pending"

/** 消息中心：聚合评论、做过、纠错消息列表。 */
@Composable
fun MessagesScreen(
    onOpenRecipe: (String) -> Unit,
    modifier: Modifier = Modifier,
    viewModel: MessagesViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    Scaffold(topBar = { WanLiTopBar(title = "消息") }) { padding ->
        WanLiPullToRefreshBox(
            isRefreshing = uiState is MessagesUiState.Loading,
            onRefresh = viewModel::refresh,
            modifier = modifier.fillMaxSize().padding(padding),
        ) {
        when (val state = uiState) {
            is MessagesUiState.Loading -> LazyColumn(
                contentPadding = PaddingValues(20.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
                modifier = Modifier.fillMaxSize(),
            ) {
                items(3) { SkeletonCard() }
            }

            is MessagesUiState.Error -> ScrollableFill {
                EmptyState(
                    icon = Icons.Outlined.ErrorOutline,
                    title = "加载失败",
                    subtitle = state.message,
                    primaryLabel = "重试",
                    onPrimary = viewModel::refresh,
                )
            }

            is MessagesUiState.Success -> if (state.items.isEmpty()) {
                ScrollableFill {
                    EmptyState(
                        icon = Icons.Outlined.Notifications,
                        title = "暂无消息",
                        subtitle = "有人评论或做过你的菜谱时会出现在这里",
                    )
                }
            } else {
                LazyColumn(
                    contentPadding = PaddingValues(horizontal = 20.dp, vertical = 12.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.fillMaxSize(),
                ) {
                    items(state.items, key = { it.id }) { item ->
                        MessageCard(item = item, onOpenRecipe = onOpenRecipe)
                    }
                }
            }
        }
        }
    }
}

@Composable
private fun MessageCard(item: MessageDto, onOpenRecipe: (String) -> Unit) {
    val (icon, tint) = messageIcon(item.type)
    WanLiCard(onClick = item.recipeVersionId?.let { versionId -> { onOpenRecipe(versionId) } }) {
        Row(
            modifier = Modifier.fillMaxWidth().cardContentPadding(),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = tint,
                modifier = Modifier.size(24.dp),
            )
            Spacer(Modifier.width(12.dp))
            Column(Modifier.weight(1f)) {
                AuthorRow(
                    nickname = messageTitle(item),
                    subtitle = if (item.type == TYPE_COOKING_RECORD) null else item.text,
                    trailing = {
                        item.createdAt?.let {
                            Text(
                                // 后端返回 RFC3339 时间戳，take(10) 取日期部分
                                text = it.take(10),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                            )
                        }
                    },
                )
                if (item.type == TYPE_COOKING_RECORD) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        RatingStars(rating = item.rating)
                        item.text?.let {
                            if (item.rating != null) Spacer(Modifier.width(8.dp))
                            Text(
                                text = it,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                maxLines = 1,
                                overflow = TextOverflow.Ellipsis,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun messageIcon(type: String): Pair<ImageVector, Color> {
    val primary = MaterialTheme.colorScheme.primary
    return when (type) {
        TYPE_COMMENT -> Icons.Outlined.ChatBubbleOutline to primary
        TYPE_COOKING_RECORD -> Icons.Outlined.Restaurant to primary
        TYPE_CORRECTION_REVIEWED -> Icons.Outlined.CheckCircle to SuccessGreen40
        TYPE_CORRECTION_PENDING -> Icons.Outlined.RateReview to WarningAmber40
        else -> Icons.Outlined.Notifications to primary
    }
}

private fun messageTitle(item: MessageDto): String = when (item.type) {
    TYPE_COMMENT -> "${item.actorNickname.ifBlank { "有人" }} 评论了《${item.recipeTitle}》"
    TYPE_COOKING_RECORD -> "${item.actorNickname.ifBlank { "有人" }} 做了《${item.recipeTitle}》"
    TYPE_CORRECTION_REVIEWED -> "《${item.recipeTitle}》纠错审核结果"
    TYPE_CORRECTION_PENDING -> "${item.actorNickname.ifBlank { "有人" }} 提交了《${item.recipeTitle}》的纠错"
    else -> item.actorNickname.ifBlank { "系统通知" }
}
