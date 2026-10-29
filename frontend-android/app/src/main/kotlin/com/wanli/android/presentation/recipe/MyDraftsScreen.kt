package com.wanli.android.presentation.recipe

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.RateReview
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.wanli.android.data.remote.RecipeDraftDto
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.theme.TomatoRed

/** 待校对解析草稿列表：「去校对」进校对页修改，「发布」原样公开。 */
@Composable
fun MyDraftsScreen(
    onBack: () -> Unit,
    onOpenProofread: (String) -> Unit,
    onPublished: (String) -> Unit,
    viewModel: MyDraftsViewModel = hiltViewModel()
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()

    (state as? MyDraftsUiState.Success)?.publishedVersionId?.let { versionId ->
        LaunchedEffect(versionId) {
            viewModel.consumePublished()
            onPublished(versionId)
        }
    }

    Scaffold(
        topBar = { WanLiTopBar(title = "待校对菜谱", onBack = onBack, modifier = Modifier.statusBarsPadding()) }
    ) { padding ->
        Box(Modifier.fillMaxSize().padding(padding)) {
            when (val current = state) {
                MyDraftsUiState.Loading -> Column(
                    modifier = Modifier.padding(20.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    repeat(3) { SkeletonCard() }
                }

                is MyDraftsUiState.Error -> EmptyState(
                    icon = Icons.Outlined.ErrorOutline,
                    title = "加载失败",
                    subtitle = current.message,
                    modifier = Modifier.align(Alignment.Center),
                    primaryLabel = "重试",
                    onPrimary = viewModel::load
                )

                is MyDraftsUiState.Success -> {
                    if (current.items.isEmpty()) {
                        EmptyState(
                            icon = Icons.Outlined.RateReview,
                            title = "没有待校对的菜谱",
                            subtitle = "导入视频解析成功后，草稿会先出现在这里，校对完成再发布到发现页",
                            modifier = Modifier.align(Alignment.Center)
                        )
                    } else {
                        WanLiPullToRefreshBox(
                            isRefreshing = false,
                            onRefresh = viewModel::load,
                            modifier = Modifier.fillMaxSize()
                        ) {
                            LazyColumn(
                                modifier = Modifier.fillMaxSize(),
                                contentPadding = PaddingValues(16.dp),
                                verticalArrangement = Arrangement.spacedBy(10.dp)
                            ) {
                                current.message?.let { message ->
                                    item {
                                        messageBanner(message, isError = current.messageIsError)
                                    }
                                }
                                items(current.items, key = { it.recipe_version_id }) { draft ->
                                    DraftCard(
                                        draft = draft,
                                        publishing = draft.recipe_version_id in current.publishingIds,
                                        onProofread = { onOpenProofread(draft.recipe_version_id) },
                                        onPublish = { viewModel.publish(draft) }
                                    )
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun messageBanner(message: String, isError: Boolean) {
    Text(
        text = message,
        style = MaterialTheme.typography.bodyMedium,
        color = if (isError) MaterialTheme.colorScheme.error else TomatoRed,
        modifier = Modifier.fillMaxWidth().padding(horizontal = 4.dp, vertical = 4.dp)
    )
}

@Composable
private fun DraftCard(
    draft: RecipeDraftDto,
    publishing: Boolean,
    onProofread: () -> Unit,
    onPublish: () -> Unit
) {
    WanLiCard(modifier = Modifier.fillMaxWidth()) {
        Column(Modifier.padding(14.dp)) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                Box(
                    modifier = Modifier
                        .size(64.dp)
                        .clip(MaterialTheme.shapes.medium)
                ) {
                    AsyncImage(
                        model = draft.cover_url,
                        contentDescription = draft.title,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier.fillMaxSize()
                    )
                }
                Spacer(Modifier.width(12.dp))
                Column(Modifier.weight(1f)) {
                    Text(
                        draft.title,
                        style = MaterialTheme.typography.titleMedium,
                        maxLines = 2,
                        overflow = TextOverflow.Ellipsis
                    )
                    Spacer(Modifier.height(4.dp))
                    Text(
                        subtitleOf(draft),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis
                    )
                }
            }
            Spacer(Modifier.height(12.dp))
            Row(horizontalArrangement = Arrangement.spacedBy(10.dp)) {
                OutlinedButton(
                    onClick = onProofread,
                    enabled = !publishing,
                    shape = RoundedCornerShape(50),
                    modifier = Modifier.weight(1f)
                ) {
                    Text("去校对")
                }
                Button(
                    onClick = onPublish,
                    enabled = !publishing,
                    shape = RoundedCornerShape(50),
                    modifier = Modifier.weight(1f)
                ) {
                    if (publishing) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(16.dp),
                            strokeWidth = 2.dp,
                            color = MaterialTheme.colorScheme.onPrimary
                        )
                        Spacer(Modifier.width(8.dp))
                        Text("发布中")
                    } else {
                        Text("直接发布")
                    }
                }
            }
        }
    }
}

private fun subtitleOf(draft: RecipeDraftDto): String {
    val expired = draft.video_status != "ACTIVE"
    return when {
        draft.bvid.isNotBlank() && expired -> "${draft.bvid} · 原视频已失效"
        draft.bvid.isNotBlank() -> draft.bvid
        expired -> "原视频已失效，菜谱内容仍可发布"
        else -> "AI 解析结果，尚未发布"
    }
}
