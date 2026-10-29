package com.wanli.android.presentation.discover

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
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
import androidx.compose.material.icons.outlined.Search
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.wanli.android.data.remote.RecipeSummaryDto
import com.wanli.android.presentation.components.AuthorRow
import com.wanli.android.presentation.components.CircleIconButton
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.ErrorMessage
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.SocialBadge
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding
import com.wanli.android.presentation.theme.CardWhite
import com.wanli.android.presentation.theme.InkPrimary
import com.wanli.android.presentation.theme.OnTomato
import com.wanli.android.presentation.theme.OutlineWarm
import com.wanli.android.presentation.theme.TomatoRed

@Composable
fun DiscoverScreen(
    onOpenRecipe: (versionId: String) -> Unit,
    modifier: Modifier = Modifier,
    viewModel: DiscoverViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Column(modifier = modifier.fillMaxSize()) {
        WanLiTopBar(
            title = "发现",
            actions = {
                CircleIconButton(onClick = {}, contentDescription = "搜索") {
                    Icon(Icons.Outlined.Search, contentDescription = null, tint = InkPrimary)
                }
            },
        )

        val currentSort = (uiState as? DiscoverUiState.Success)?.sort ?: viewModel.currentSort
        Row(
            modifier = Modifier.fillMaxWidth().padding(horizontal = 20.dp, vertical = 4.dp),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            SortChip("最新", currentSort == DiscoverViewModel.SORT_LATEST) {
                viewModel.setSort(DiscoverViewModel.SORT_LATEST)
            }
            SortChip("最热", currentSort == DiscoverViewModel.SORT_HOTTEST) {
                viewModel.setSort(DiscoverViewModel.SORT_HOTTEST)
            }
        }

        WanLiPullToRefreshBox(
            isRefreshing = uiState is DiscoverUiState.Loading,
            onRefresh = { viewModel.refresh() },
            modifier = Modifier.fillMaxSize().weight(1f),
        ) {
            when (val state = uiState) {
                is DiscoverUiState.Loading -> LazyVerticalStaggeredGrid(
                    columns = StaggeredGridCells.Fixed(2),
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
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
                        TextButton(onClick = { viewModel.refresh() }) { Text("重试") }
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
                            contentPadding = PaddingValues(horizontal = 20.dp, vertical = 8.dp),
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

internal fun discoverItemKey(item: RecipeSummaryDto): String = item.recipe_version_id

@Composable
private fun SortChip(label: String, selected: Boolean, onClick: () -> Unit) {
    FilterChip(
        selected = selected,
        onClick = onClick,
        label = { Text(label, style = MaterialTheme.typography.labelMedium) },
        shape = MaterialTheme.shapes.extraLarge,
        colors = FilterChipDefaults.filterChipColors(
            containerColor = CardWhite,
            labelColor = InkPrimary,
            selectedContainerColor = TomatoRed,
            selectedLabelColor = OnTomato,
        ),
        border = FilterChipDefaults.filterChipBorder(
            enabled = true,
            selected = selected,
            borderColor = OutlineWarm,
            selectedBorderColor = TomatoRed,
        ),
    )
}

/** 双列瀑布流卡片：图片按 id 取三种比例错落，卡片只留标题与作者。 */
@Composable
private fun DiscoverCard(recipe: RecipeSummaryDto, onClick: () -> Unit) {
    val ratios = listOf(4f / 5f, 1f, 3f / 4f)
    val ratio = ratios[(recipe.recipe_version_id.hashCode() and 0x7FFFFFFF) % ratios.size]
    WanLiCard(onClick = onClick) {
        if (recipe.cover_url != null) {
            AsyncImage(
                model = recipe.cover_url,
                contentDescription = null,
                modifier = Modifier.fillMaxWidth().aspectRatio(ratio),
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
            Spacer(Modifier.height(8.dp))
            AuthorRow(
                nickname = recipe.author_nickname.ifBlank { recipe.up_nickname },
                avatarUrl = null,
                trailing = {
                    when {
                        recipe.cookingRecordCount > 0 -> SocialBadge("★ ${recipe.cookingRecordCount}")
                        recipe.public_version_count > 1 -> SocialBadge("${recipe.public_version_count} 版本")
                    }
                },
            )
        }
    }
}
