package com.wanli.android.presentation.home

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.SoupKitchen
import androidx.compose.material3.ButtonDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.wanli.android.data.remote.CookingFeedItemDto
import com.wanli.android.presentation.components.AuthorRow
import com.wanli.android.presentation.components.CircleIconButton
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.ErrorMessage
import com.wanli.android.presentation.components.RatingStars
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding
import com.wanli.android.presentation.theme.InkSecondary
import com.wanli.android.presentation.theme.TomatoRed

@Composable
fun HomeScreen(
    onOpenRecipe: (String) -> Unit,
    onOpenMessages: () -> Unit,
    onOpenDiscover: () -> Unit,
    modifier: Modifier = Modifier,
    onNavigateToLogin: () -> Unit = {},
    onProofread: (String) -> Unit = {},
    onOpenImport: () -> Unit = {},
    homeViewModel: HomeViewModel = hiltViewModel(),
) {
    val feedState by homeViewModel.feedState.collectAsStateWithLifecycle()

    Column(modifier = modifier.fillMaxSize()) {
        WanLiTopBar(
            title = "碗里",
            actions = {
                CircleIconButton(onClick = onOpenMessages, contentDescription = "消息") {
                    Icon(Icons.Outlined.Notifications, contentDescription = null, tint = TomatoRed)
                }
            },
        )
        WanLiPullToRefreshBox(
            isRefreshing = feedState is HomeFeedUiState.Loading,
            onRefresh = homeViewModel::refresh,
            modifier = Modifier.fillMaxSize(),
        ) {
            LazyColumn(
                modifier = Modifier.fillMaxSize(),
                contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
            ) {
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
                                    title = "还没有人做过菜",
                                    subtitle = "导入第一道菜谱，或者去发现页挑一道跟着做",
                                    primaryLabel = "导入菜谱",
                                    onPrimary = onOpenImport,
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
}

@Composable
private fun FeedCard(item: CookingFeedItemDto, onClick: () -> Unit) {
    WanLiCard(onClick = onClick) {
        val imageUrl = item.photoUrl ?: item.coverUrl
        if (imageUrl != null) {
            AsyncImage(
                model = imageUrl,
                contentDescription = null,
                modifier = Modifier.fillMaxWidth().aspectRatio(4f / 3f),
                contentScale = ContentScale.Crop,
            )
        }
        Column(Modifier.cardContentPadding()) {
            AuthorRow(
                nickname = item.authorNickname.ifBlank { "碗友" },
                subtitle = "做了《${item.recipeTitle}》",
                trailing = { RatingStars(rating = item.rating) },
            )
            Spacer(Modifier.height(8.dp))
            Text(item.recipeTitle, style = MaterialTheme.typography.titleMedium)
            if (!item.feedback.isNullOrBlank()) {
                Spacer(Modifier.height(4.dp))
                Text(
                    item.feedback,
                    style = MaterialTheme.typography.bodyMedium,
                    color = InkSecondary,
                )
            }
            Spacer(Modifier.height(12.dp))
            Row(verticalAlignment = Alignment.CenterVertically) {
                Spacer(Modifier.weight(1f))
                OutlinedButton(
                    onClick = onClick,
                    shape = CircleShape,
                    contentPadding = PaddingValues(horizontal = 16.dp, vertical = 6.dp),
                    colors = ButtonDefaults.outlinedButtonColors(contentColor = TomatoRed),
                ) {
                    Text("跟着做", style = MaterialTheme.typography.labelSmall)
                }
            }
        }
    }
}
