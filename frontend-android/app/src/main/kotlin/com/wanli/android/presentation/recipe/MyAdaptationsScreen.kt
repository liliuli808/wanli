package com.wanli.android.presentation.recipe

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.RamenDining
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Switch
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.WanLiTopBar

@Composable
fun MyAdaptationsScreen(
    onBack: () -> Unit,
    onOpenVersion: (String) -> Unit,
    viewModel: MyAdaptationsViewModel = hiltViewModel()
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    Scaffold(
        topBar = { WanLiTopBar(title = "我的适配版", onBack = onBack) }
    ) { padding ->
        Box(Modifier.fillMaxSize().padding(padding)) {
            when (val current = state) {
                MyAdaptationsUiState.Loading -> Column(
                    modifier = Modifier.padding(20.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    repeat(4) { SkeletonCard() }
                }
                is MyAdaptationsUiState.Error -> EmptyState(
                    icon = Icons.Outlined.ErrorOutline,
                    title = "加载失败",
                    subtitle = current.message,
                    modifier = Modifier.align(Alignment.Center),
                    primaryLabel = "重试",
                    onPrimary = viewModel::load
                )
                is MyAdaptationsUiState.Success -> {
                    if (current.items.isEmpty()) {
                        EmptyState(
                            icon = Icons.Outlined.RamenDining,
                            title = "还没有个人适配版",
                            subtitle = "在菜谱详情页点「适配我的厨房」试试",
                            modifier = Modifier.align(Alignment.Center)
                        )
                    } else {
                        LazyColumn(
                            modifier = Modifier.fillMaxSize(),
                            contentPadding = PaddingValues(16.dp),
                            verticalArrangement = Arrangement.spacedBy(10.dp)
                        ) {
                            current.message?.let { message ->
                                item {
                                    Text(message, color = MaterialTheme.colorScheme.error)
                                }
                            }
                            items(current.items, key = { it.id }) { item ->
                                Card(
                                    elevation = CardDefaults.cardElevation(defaultElevation = 1.dp),
                                    modifier = Modifier.fillMaxWidth().clickable {
                                        onOpenVersion(item.id)
                                    }
                                ) {
                                    Row(
                                        modifier = Modifier.fillMaxWidth().padding(14.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        Column(modifier = Modifier.weight(1f)) {
                                            Text(item.title, style = MaterialTheme.typography.titleMedium)
                                            Text(
                                                if (item.is_public) "公开分享中" else "仅自己可见",
                                                style = MaterialTheme.typography.bodySmall,
                                                color = MaterialTheme.colorScheme.onSurfaceVariant
                                            )
                                        }
                                        Switch(
                                            checked = item.is_public,
                                            enabled = item.id !in current.updatingIds,
                                            onCheckedChange = {
                                                viewModel.setVisibility(item.id, it)
                                            }
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
}
