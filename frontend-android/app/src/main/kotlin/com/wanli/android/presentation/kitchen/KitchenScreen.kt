package com.wanli.android.presentation.kitchen

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Kitchen
import androidx.compose.material.icons.outlined.ShoppingBag
import androidx.compose.material3.Button
import androidx.compose.material3.FilterChip
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.presentation.components.ErrorMessage
import com.wanli.android.presentation.components.SectionHeader
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiFilterChip
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding

private val COMMON_INGREDIENTS = listOf("盐", "生抽", "老抽", "醋", "料酒", "糖", "大蒜", "姜", "葱", "花椒", "八角", "辣椒粉")
private val COMMON_TOOLS = listOf("炒锅", "电饭煲", "空气炸锅", "烤箱", "微波炉", "蒸锅", "高压锅")

@Composable
fun KitchenScreen(
    onBack: () -> Unit = {},
    viewModel: KitchenViewModel = androidx.hilt.navigation.compose.hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = { WanLiTopBar(title = "我的厨房", onBack = onBack) }
    ) { padding ->
        WanLiPullToRefreshBox(
            isRefreshing = uiState is KitchenUiState.Loading,
            onRefresh = { viewModel.loadKitchen() },
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(16.dp)
                .verticalScroll(rememberScrollState())
        ) {
            when (val state = uiState) {
                is KitchenUiState.Loading -> {
                    SkeletonCard(modifier = Modifier.padding(top = 16.dp))
                    SkeletonCard(modifier = Modifier.padding(top = 16.dp))
                }
                is KitchenUiState.Error -> {
                    ErrorMessage(message = state.message)
                    Button(
                        onClick = { viewModel.loadKitchen() },
                        modifier = Modifier.padding(top = 16.dp)
                    ) {
                        Text("重试")
                    }
                }
                is KitchenUiState.Success -> {
                    Text(
                        text = "勾选你厨房里现有的调料和厨具，适配菜谱时会更准确",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    ChipSection(
                        icon = Icons.Outlined.ShoppingBag,
                        title = "拥有的调料",
                        options = COMMON_INGREDIENTS,
                        selected = state.ingredients,
                        onSelectionChanged = { selected ->
                            viewModel.saveKitchen(selected, state.tools)
                        }
                    )

                    ChipSection(
                        icon = Icons.Outlined.Kitchen,
                        title = "拥有的厨具",
                        options = COMMON_TOOLS,
                        selected = state.tools,
                        onSelectionChanged = { selected ->
                            viewModel.saveKitchen(state.ingredients, selected)
                        }
                    )
                }
            }
        }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun ChipSection(
    icon: ImageVector,
    title: String,
    options: List<String>,
    selected: List<String>,
    onSelectionChanged: (List<String>) -> Unit
) {
    WanLiCard(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 16.dp)
    ) {
        Column(modifier = Modifier.cardContentPadding()) {
            SectionHeader(title = title, icon = icon)
            FlowRow(modifier = Modifier.fillMaxWidth().padding(top = 8.dp)) {
                options.forEach { option ->
                    val isSelected = option in selected
                    WanLiFilterChip(
                        selected = isSelected,
                        onClick = {
                            val newSelection = if (isSelected) {
                                selected - option
                            } else {
                                selected + option
                            }
                            onSelectionChanged(newSelection)
                        },
                        label = option,
                        modifier = Modifier.padding(end = 8.dp, bottom = 4.dp)
                    )
                }
            }
        }
    }
}
