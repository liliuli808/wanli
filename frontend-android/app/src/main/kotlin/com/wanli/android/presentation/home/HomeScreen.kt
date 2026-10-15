package com.wanli.android.presentation.home

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.SoupKitchen
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.wanli.android.data.remote.CookingFeedItemDto
import com.wanli.android.presentation.components.AuthorRow
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.ErrorMessage
import com.wanli.android.presentation.components.ImportBar
import com.wanli.android.presentation.components.RatingStars
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.StatusBanner
import com.wanli.android.presentation.components.StatusBannerType
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.cardContentPadding

@Composable
fun HomeScreen(
    onOpenRecipe: (String) -> Unit,
    onOpenMessages: () -> Unit,
    onOpenDiscover: () -> Unit,
    modifier: Modifier = Modifier,
    onNavigateToLogin: () -> Unit = {},
    onProofread: (String) -> Unit = {},
    homeViewModel: HomeViewModel = hiltViewModel(),
    importViewModel: ImportViewModel = hiltViewModel(),
) {
    val feedState by homeViewModel.feedState.collectAsStateWithLifecycle()
    val importState by importViewModel.uiState.collectAsStateWithLifecycle()
    var urlInput by rememberSaveable { mutableStateOf("") }

    WanLiPullToRefreshBox(
        isRefreshing = feedState is HomeFeedUiState.Loading,
        onRefresh = homeViewModel::refresh,
        modifier = modifier.fillMaxSize(),
    ) {
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 20.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text(
                    text = "碗里",
                    style = MaterialTheme.typography.headlineMedium,
                    modifier = Modifier.weight(1f),
                )
                IconButton(onClick = onOpenMessages) {
                    Icon(
                        Icons.Outlined.Notifications,
                        contentDescription = "消息",
                        tint = MaterialTheme.colorScheme.primary,
                    )
                }
            }
        }
        item {
            ImportBar(
                url = urlInput,
                onUrlChange = { urlInput = it },
                onSubmit = { importViewModel.submit(urlInput) },
            )
        }
        item {
            ImportStatusSection(
                importState = importState,
                onOpenRecipe = onOpenRecipe,
                onProofread = onProofread,
                onNavigateToLogin = onNavigateToLogin,
                onRetry = { importViewModel.submit(urlInput) },
                onReset = { importViewModel.reset(); urlInput = "" },
            )
        }
        when (val state = feedState) {
            is HomeFeedUiState.Loading -> items(3) { SkeletonCard() }
            is HomeFeedUiState.Error -> item {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    ErrorMessage(message = state.message)
                    TextButton(onClick = homeViewModel::refresh) { Text("重试") }
                }
            }
            is HomeFeedUiState.Success -> {
                if (state.items.isEmpty()) {
                    item {
                        EmptyState(
                            icon = Icons.Outlined.SoupKitchen,
                            title = "还没有菜谱动态",
                            subtitle = "导入第一道菜谱，或去看看社区里大家都在做什么",
                            primaryLabel = "粘贴链接导入",
                            onPrimary = { importViewModel.submit(urlInput) },
                            secondaryLabel = "去发现页逛逛",
                            onSecondary = onOpenDiscover,
                        )
                    }
                } else {
                    items(state.items, key = { it.id }) { item ->
                        FeedCard(item = item, onClick = { onOpenRecipe(item.recipeVersionId) })
                    }
                }
            }
        }
    }
    }
}

@Composable
private fun FeedCard(item: CookingFeedItemDto, onClick: () -> Unit) {
    WanLiCard(onClick = onClick) {
        val imageUrl = item.photoUrl ?: item.coverUrl
        if (imageUrl != null) {
            AsyncImage(
                model = imageUrl,
                contentDescription = null,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(180.dp)
                    .clip(MaterialTheme.shapes.medium),
                contentScale = ContentScale.Crop,
            )
        }
        Column(Modifier.cardContentPadding()) {
            AuthorRow(
                nickname = item.authorNickname,
                subtitle = "做了《${item.recipeTitle}》",
                trailing = { RatingStars(rating = item.rating) },
            )
            if (!item.feedback.isNullOrBlank()) {
                Spacer(Modifier.height(8.dp))
                Text(item.feedback, style = MaterialTheme.typography.bodyMedium)
            }
        }
    }
}

/** 解析任务各状态的语义化反馈区。 */
@Composable
private fun ImportStatusSection(
    importState: ImportUiState,
    onOpenRecipe: (String) -> Unit,
    onProofread: (String) -> Unit,
    onNavigateToLogin: () -> Unit,
    onRetry: () -> Unit,
    onReset: () -> Unit
) {
    when (val state = importState) {
        is ImportUiState.Idle -> Unit
        is ImportUiState.Submitting -> StatusBanner(type = StatusBannerType.INFO) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                CircularProgressIndicator(
                    strokeWidth = 2.dp,
                    modifier = Modifier.size(16.dp)
                )
                Text("正在提交解析任务…", modifier = Modifier.padding(start = 8.dp))
            }
        }
        is ImportUiState.Polling -> StatusBanner(type = StatusBannerType.INFO) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                CircularProgressIndicator(
                    strokeWidth = 2.dp,
                    modifier = Modifier.size(16.dp)
                )
                Text(state.message, modifier = Modifier.padding(start = 8.dp))
            }
        }
        is ImportUiState.Completed -> StatusBanner(type = StatusBannerType.SUCCESS) {
            Text(
                text = "解析完成！已生成待校对菜谱草稿",
                style = MaterialTheme.typography.titleSmall
            )
            BannerActions {
                state.recipeVersionId?.let { versionId ->
                    Button(onClick = { onProofread(versionId) }) {
                        Text("去校对发布")
                    }
                }
                TextButton(onClick = onReset) {
                    Text("继续解析下一个")
                }
            }
        }
        is ImportUiState.Failed -> StatusBanner(type = StatusBannerType.ERROR) {
            Text(text = state.message, style = MaterialTheme.typography.bodyMedium)
            BannerActions {
                OutlinedButton(onClick = onRetry) {
                    Text("重试")
                }
                TextButton(onClick = onReset) {
                    Text("关闭")
                }
            }
        }
        is ImportUiState.Duplicate -> StatusBanner(type = StatusBannerType.WARNING) {
            Text(
                text = "该视频已有解析菜谱",
                style = MaterialTheme.typography.titleSmall
            )
            BannerActions {
                // AC1：重复 BVID 直接跳转已有菜谱详情页
                state.recipeVersionId?.let { versionId ->
                    Button(onClick = { onOpenRecipe(versionId) }) {
                        Text("查看菜谱")
                    }
                }
                TextButton(onClick = onReset) {
                    Text("知道了")
                }
            }
        }
        is ImportUiState.LoginRequired -> StatusBanner(type = StatusBannerType.WARNING) {
            Text("登录后才能解析视频", style = MaterialTheme.typography.bodyMedium)
            BannerActions {
                Button(onClick = onNavigateToLogin) {
                    Text("去登录")
                }
            }
        }
    }
}

@Composable
private fun BannerActions(content: @Composable () -> Unit) {
    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        modifier = Modifier.padding(top = 8.dp)
    ) {
        content()
    }
}
