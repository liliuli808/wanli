package com.wanli.android.presentation.discover

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.staggeredgrid.LazyVerticalStaggeredGrid
import androidx.compose.foundation.lazy.staggeredgrid.StaggeredGridCells
import androidx.compose.foundation.lazy.staggeredgrid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Restaurant
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
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
import com.wanli.android.data.remote.RecipeSummaryDto
import com.wanli.android.presentation.components.AuthorRow
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.ErrorMessage
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.SocialBadge
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding

@Composable
fun DiscoverScreen(
    onOpenRecipe: (versionId: String) -> Unit,
    modifier: Modifier = Modifier,
    viewModel: DiscoverViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        modifier = modifier,
        topBar = { WanLiTopBar(title = "发现好菜") }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            val currentSort = (uiState as? DiscoverUiState.Success)?.sort ?: viewModel.currentSort
            val selectedTab = if (currentSort == DiscoverViewModel.SORT_HOTTEST) 1 else 0
            TabRow(
                selectedTabIndex = selectedTab,
                containerColor = MaterialTheme.colorScheme.background
            ) {
                Tab(
                    selected = selectedTab == 0,
                    onClick = { viewModel.setSort(DiscoverViewModel.SORT_LATEST) },
                    text = { Text("最新") }
                )
                Tab(
                    selected = selectedTab == 1,
                    onClick = { viewModel.setSort(DiscoverViewModel.SORT_HOTTEST) },
                    text = { Text("最热") }
                )
            }

            WanLiPullToRefreshBox(
                isRefreshing = uiState is DiscoverUiState.Loading,
                onRefresh = { viewModel.refresh() },
                modifier = Modifier
                    .fillMaxSize()
                    .weight(1f),
            ) {
            when (val state = uiState) {
                is DiscoverUiState.Loading -> LazyVerticalStaggeredGrid(
                    columns = StaggeredGridCells.Fixed(2),
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalItemSpacing = 12.dp,
                ) {
                    items(4) { SkeletonCard() }
                }
                is DiscoverUiState.Error -> Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        ErrorMessage(message = state.message)
                        TextButton(onClick = { viewModel.refresh() }) {
                            Text("重试")
                        }
                    }
                }
                is DiscoverUiState.Success -> {
                    if (state.items.isEmpty()) {
                        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
                            EmptyState(
                                icon = Icons.Outlined.Restaurant,
                                title = "还没有公开菜谱",
                                subtitle = "去首页解析一个视频，装第一碗吧",
                            )
                        }
                    } else {
                        LazyVerticalStaggeredGrid(
                            columns = StaggeredGridCells.Fixed(2),
                            modifier = Modifier.fillMaxSize(),
                            contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
                            horizontalArrangement = Arrangement.spacedBy(12.dp),
                            verticalItemSpacing = 12.dp,
                        ) {
                            items(state.items, key = ::discoverItemKey) { recipe ->
                                DiscoverCard(
                                    recipe = recipe,
                                    onClick = { onOpenRecipe(recipe.recipe_version_id) }
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

internal fun discoverItemKey(item: RecipeSummaryDto): String = item.recipe_version_id

@Composable
private fun DiscoverCard(recipe: RecipeSummaryDto, onClick: () -> Unit) {
    WanLiCard(onClick = onClick) {
        if (recipe.cover_url != null) {
            AsyncImage(
                model = recipe.cover_url,
                contentDescription = null,
                modifier = Modifier
                    .fillMaxWidth()
                    .aspectRatio(1f)
                    .clip(MaterialTheme.shapes.medium),
                contentScale = ContentScale.Crop,
            )
        }
        Column(Modifier.cardContentPadding()) {
            Text(
                recipe.title,
                style = MaterialTheme.typography.titleSmall,
                maxLines = 2,
                overflow = TextOverflow.Ellipsis,
            )
            Spacer(Modifier.height(6.dp))
            AuthorRow(
                nickname = recipe.author_nickname.ifBlank { recipe.up_nickname },
                trailing = {
                    when {
                        recipe.cookingRecordCount > 0 -> SocialBadge("${recipe.cookingRecordCount} 人做过")
                        recipe.public_version_count > 1 -> SocialBadge("${recipe.public_version_count} 个版本")
                    }
                },
            )
        }
    }
}
