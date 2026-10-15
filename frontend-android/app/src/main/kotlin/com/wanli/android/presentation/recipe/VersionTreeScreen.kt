package com.wanli.android.presentation.recipe

import androidx.compose.foundation.background
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
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyListScope
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.AccountTree
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.data.remote.VersionTreeNodeDto
import com.wanli.android.presentation.components.AuthorRow
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.ScrollableFill
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.SocialBadge
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding
import com.wanli.android.presentation.theme.OutlineVariantGray

/** 递归渲染时的最大深度上限，防止异常数据导致无限展开。 */
private const val MAX_TREE_DEPTH = 12

@Composable
fun VersionTreeScreen(
    onBack: () -> Unit,
    onOpenVersion: (String) -> Unit,
    viewModel: VersionTreeViewModel = hiltViewModel()
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    Scaffold(
        topBar = { WanLiTopBar(title = "版本派生树", onBack = onBack) }
    ) { padding ->
        WanLiPullToRefreshBox(
            isRefreshing = state is VersionTreeUiState.Loading,
            onRefresh = viewModel::load,
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
            when (val current = state) {
                VersionTreeUiState.Loading -> Column(
                    modifier = Modifier.padding(20.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    repeat(4) { SkeletonCard() }
                }
                is VersionTreeUiState.Error -> ScrollableFill {
                    EmptyState(
                        icon = Icons.Outlined.ErrorOutline,
                        title = "加载失败",
                        subtitle = current.message,
                        primaryLabel = "重试",
                        onPrimary = viewModel::load
                    )
                }
                is VersionTreeUiState.Success -> {
                    if (current.nodes.isEmpty()) {
                        ScrollableFill {
                            EmptyState(
                                icon = Icons.Outlined.AccountTree,
                                title = "暂无公开版本",
                                subtitle = "这个菜谱还没有可查看的版本谱系",
                            )
                        }
                    } else {
                        VersionTree(current.nodes, onOpenVersion)
                    }
                }
            }
        }
    }
}

/** 谱系树：根节点大卡置顶，后代按深度缩进并以竖线连接。 */
@Composable
private fun VersionTree(nodes: List<VersionTreeNodeDto>, onOpenVersion: (String) -> Unit) {
    // created_at 假定为 ISO-8601 UTC 格式，字典序排序即时间序。
    val childrenByParent = remember(nodes) {
        nodes.groupBy { it.parent_id }.mapValues { (_, v) -> v.sortedBy { it.created_at ?: "" } }
    }
    fun descendantCount(id: String): Int {
        var count = 0
        var frontier = childrenByParent[id].orEmpty()
        val visited = frontier.mapTo(mutableSetOf()) { it.id }
        while (frontier.isNotEmpty()) {
            count += frontier.size
            frontier = frontier.flatMap { childrenByParent[it.id].orEmpty() }.filter { visited.add(it.id) }
        }
        return count
    }

    val roots = childrenByParent[null].orEmpty()
    if (roots.isEmpty()) {
        // 数据全是孤儿节点（parent_id 悬空）时兜底空态，避免白屏。
        Box(Modifier.fillMaxSize()) {
            EmptyState(
                icon = Icons.Outlined.AccountTree,
                title = "暂无公开版本",
                subtitle = "这个菜谱还没有可查看的版本谱系",
                modifier = Modifier.align(Alignment.Center)
            )
        }
        return
    }
    LazyColumn(
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(20.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        roots.forEach { root ->
            item(key = root.id) {
                VersionNodeCard(
                    node = root,
                    depth = 0,
                    badgeText = descendantCount(root.id).takeIf { it > 0 }?.let { "$it 个改编" }
                        ?: "UP 主原版",
                    onClick = { onOpenVersion(root.id) }
                )
            }
            renderChildren(
                parentId = root.id,
                depth = 1,
                childrenByParent = childrenByParent,
                onOpenVersion = onOpenVersion
            )
        }
    }
}

/** 递归展开子节点；visited 防止环路数据造成死循环。 */
private fun LazyListScope.renderChildren(
    parentId: String,
    depth: Int,
    childrenByParent: Map<String?, List<VersionTreeNodeDto>>,
    onOpenVersion: (String) -> Unit,
    visited: MutableSet<String> = mutableSetOf()
) {
    if (depth > MAX_TREE_DEPTH) return
    childrenByParent[parentId].orEmpty().forEach { child ->
        if (!visited.add(child.id)) return@forEach
        item(key = child.id) {
            VersionNodeCard(
                node = child,
                depth = depth,
                badgeText = versionTypeLabel(child.version_type),
                onClick = { onOpenVersion(child.id) }
            )
        }
        renderChildren(child.id, depth + 1, childrenByParent, onOpenVersion, visited)
    }
}

/** 单个版本节点：深度缩进 + 竖线连接符（depth > 0）+ 标题/作者/变更说明/徽章。 */
@Composable
private fun VersionNodeCard(
    node: VersionTreeNodeDto,
    depth: Int,
    badgeText: String,
    onClick: () -> Unit
) {
    // 不合并后代语义：卡片自身可点击，仅在 Row 上补充层级信息供 TalkBack 朗读。
    val depthLabel = if (depth == 0) "UP 主原版" else "第 ${depth + 1} 代版本"
    Row(
        Modifier
            .fillMaxWidth()
            .semantics { contentDescription = "${node.title}，$depthLabel" },
        verticalAlignment = Alignment.CenterVertically
    ) {
        if (depth > 0) {
            Spacer(Modifier.width((minOf(depth, 4) * 20).dp))
            Box(
                Modifier
                    .width(2.dp)
                    .height(40.dp)
                    .background(OutlineVariantGray)
            )
            Spacer(Modifier.width(8.dp))
        }
        WanLiCard(onClick = onClick, modifier = Modifier.weight(1f)) {
            Column(Modifier.cardContentPadding()) {
                Text(
                    node.title,
                    style = MaterialTheme.typography.titleMedium,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis
                )
                Spacer(Modifier.height(8.dp))
                AuthorRow(
                    nickname = node.author_nickname.ifBlank { "佚名" },
                    subtitle = node.change_log?.takeIf { it.isNotBlank() },
                    trailing = { SocialBadge(badgeText) }
                )
            }
        }
    }
}
