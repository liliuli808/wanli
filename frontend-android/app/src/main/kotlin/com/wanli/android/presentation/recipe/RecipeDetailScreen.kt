package com.wanli.android.presentation.recipe

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.heightIn
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.outlined.AccountTree
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.wanli.android.data.remote.RecipeVersionData
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.ScrollableFill
import com.wanli.android.presentation.components.SectionHeader
import com.wanli.android.presentation.components.StatusBanner
import com.wanli.android.presentation.components.StatusBannerType
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar

@Composable
fun RecipeDetailScreen(
    onBack: () -> Unit,
    onCorrection: (String) -> Unit = {},
    onFork: (String) -> Unit = {},
    onAdapt: (String) -> Unit = {},
    onVersionTree: (String) -> Unit = {},
    viewModel: RecipeDetailViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = { WanLiTopBar(title = "菜谱详情", onBack = onBack) }
    ) { padding ->
        WanLiPullToRefreshBox(
            isRefreshing = uiState is RecipeDetailUiState.Loading,
            onRefresh = { viewModel.load() },
            modifier = Modifier
                .fillMaxSize()
                .padding(padding),
        ) {
            when (val state = uiState) {
                is RecipeDetailUiState.Loading -> Box(Modifier.fillMaxSize()) {
                    CircularProgressIndicator(
                        modifier = Modifier.align(Alignment.Center)
                    )
                }
                is RecipeDetailUiState.Error -> ScrollableFill {
                    EmptyState(
                        icon = Icons.Outlined.ErrorOutline,
                        title = "加载失败",
                        subtitle = state.message,
                        primaryLabel = "重试",
                        onPrimary = { viewModel.load() },
                    )
                }
                is RecipeDetailUiState.Success -> RecipeDetailContent(
                    detail = state.detail,
                    onCorrection = onCorrection,
                    onFork = onFork,
                    onAdapt = onAdapt,
                    onVersionTree = onVersionTree
                )
            }
        }
    }
}

@Composable
private fun RecipeDetailContent(
    detail: RecipeVersionData,
    onCorrection: (String) -> Unit,
    onFork: (String) -> Unit,
    onAdapt: (String) -> Unit,
    onVersionTree: (String) -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
    ) {
        if (!detail.cover_url.isNullOrBlank()) {
            AsyncImage(
                model = detail.cover_url,
                contentDescription = null,
                contentScale = ContentScale.Crop,
                modifier = Modifier
                    .fillMaxWidth()
                    .aspectRatio(16f / 9f)
                    .clip(RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp))
            )
        }

        Column(modifier = Modifier.padding(16.dp)) {
            // 视频失效警示条（AC4）
            if (detail.video_status == "EXPIRED") {
                StatusBanner(type = StatusBannerType.WARNING) {
                    Text(
                        text = detail.video_warning ?: "原视频已下架或失效，菜谱内容仍可正常浏览",
                        style = MaterialTheme.typography.bodySmall
                    )
                }
            }

            Text(
                detail.title,
                style = MaterialTheme.typography.headlineMedium,
                modifier = Modifier.padding(top = 8.dp)
            )

            // meta 行：UP / 份量 / 时长
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically,
                modifier = Modifier.padding(top = 8.dp)
            ) {
                if (detail.up_nickname.isNotBlank()) {
                    MetaChip("UP：${detail.up_nickname}")
                }
                detail.servings?.let { MetaChip(it) }
                detail.estimated_minutes?.let { MetaChip("约${it}分钟") }
            }

            // 版本谱系入口
            Surface(
                onClick = { onVersionTree(detail.recipe_id) },
                shape = MaterialTheme.shapes.small,
                color = MaterialTheme.colorScheme.secondaryContainer,
                modifier = Modifier
                    .padding(top = 12.dp)
                    .heightIn(min = 48.dp)
            ) {
                Row(
                    Modifier.padding(horizontal = 12.dp, vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Icon(
                        Icons.Outlined.AccountTree,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary
                    )
                    Spacer(Modifier.width(6.dp))
                    Text(
                        versionTypeLabel(detail.version_type),
                        style = MaterialTheme.typography.labelLarge
                    )
                    Icon(
                        Icons.AutoMirrored.Filled.KeyboardArrowRight,
                        contentDescription = null
                    )
                }
            }
            Text(
                text = "by ${detail.author_nickname.ifBlank { "佚名" }}" +
                    if (!detail.is_public && detail.version_type == "KITCHEN_FIT") "（仅自己可见）"
                    else if (!detail.is_public) "（待校对草稿）"
                    else "",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(top = 6.dp)
            )
            detail.change_log?.takeIf { it.isNotBlank() }?.let {
                Text(
                    "改动说明：$it",
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(top = 4.dp)
                )
            }

            detail.adaptation_notes?.let { notes ->
                AdaptationListCard("食材替换", notes.substitutions.map {
                    "${it.original} → ${it.replacement}：${it.reason}"
                })
                AdaptationListCard("仍缺少的食材", notes.missing_ingredients)
                AdaptationListCard("仍缺少的厨具", notes.missing_tools)
                AdaptationListCard("口味变化", notes.flavor_changes)
            }

            if (!detail.instructions.isNullOrBlank()) {
                Card(
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.tertiaryContainer
                    ),
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 12.dp)
                ) {
                    Text(
                        text = "小贴士：${detail.instructions}",
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onTertiaryContainer,
                        modifier = Modifier.padding(12.dp)
                    )
                }
            }

            SectionHeader(
                "食材",
                modifier = Modifier.padding(top = 8.dp)
            )
            Card(
                elevation = CardDefaults.cardElevation(defaultElevation = 1.dp),
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(horizontal = 16.dp, vertical = 8.dp)) {
                    CATEGORY_ORDER.forEach { category ->
                        val group = detail.ingredients.filter { it.category == category }
                        if (group.isNotEmpty()) {
                            Text(
                                categoryLabel(category),
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                modifier = Modifier.padding(top = 8.dp)
                            )
                            group.forEach { ingredient ->
                                Row(
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(vertical = 3.dp)
                                ) {
                                    Text(
                                        ingredient.name +
                                            (if (!ingredient.is_explicit) "（AI 推测）" else "") +
                                            (ingredient.substitute_for?.let { "（替代 $it）" } ?: ""),
                                        style = MaterialTheme.typography.bodyMedium
                                    )
                                    Text(
                                        ingredient.amount ?: "适量",
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                }
                            }
                        }
                    }
                }
            }

            SectionHeader(
                "步骤",
                modifier = Modifier.padding(top = 8.dp)
            )
            detail.steps.forEach { step ->
                Row(modifier = Modifier.padding(vertical = 6.dp)) {
                    Box(
                        contentAlignment = Alignment.Center,
                        modifier = Modifier
                            .size(24.dp)
                            .background(
                                color = MaterialTheme.colorScheme.primaryContainer,
                                shape = CircleShape
                            )
                    ) {
                        Text(
                            "${step.step_order}",
                            style = MaterialTheme.typography.labelSmall,
                            color = MaterialTheme.colorScheme.onPrimaryContainer
                        )
                    }
                    Column(modifier = Modifier.padding(start = 12.dp)) {
                        Text(step.description, style = MaterialTheme.typography.bodyMedium)
                        val stepMeta = buildList {
                            step.heat?.let { add("火候：$it") }
                            step.duration_seconds?.let { add("约${it}秒") }
                        }.joinToString(" · ")
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(6.dp),
                            modifier = Modifier.padding(top = 2.dp)
                        ) {
                            step.video_timestamp?.let {
                                Surface(
                                    shape = MaterialTheme.shapes.small,
                                    color = MaterialTheme.colorScheme.secondaryContainer
                                ) {
                                    Text(
                                        text = "视频 ${formatTimestamp(it)}",
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.onSecondaryContainer,
                                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
                                    )
                                }
                            }
                            if (stepMeta.isNotBlank()) {
                                Text(
                                    stepMeta,
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                        }
                    }
                }
            }

            // 操作行：等宽 OutlinedButton
            val canDerive = detail.is_public && detail.version_type != "KITCHEN_FIT"
            val actions = buildList<Pair<String, () -> Unit>> {
                if (canDerive) {
                    add("适配我的厨房" to { onAdapt(detail.id) })
                }
                if (detail.is_public && detail.version_type == "SOURCE") {
                    add("提交纠错" to { onCorrection(detail.id) })
                }
                if (canDerive) {
                    add("派生新版本" to { onFork(detail.id) })
                }
            }
            if (actions.isNotEmpty()) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(top = 18.dp, bottom = 20.dp)
                ) {
                    actions.forEach { (label, action) ->
                        OutlinedButton(
                            onClick = action,
                            modifier = Modifier.weight(1f)
                        ) {
                            Text(label, style = MaterialTheme.typography.labelMedium)
                        }
                    }
                }
            }

            if (detail.is_public) {
                CommunitySection()
            }
        }
    }
}

@Composable
private fun MetaChip(text: String) {
    Surface(
        shape = MaterialTheme.shapes.small,
        color = MaterialTheme.colorScheme.surfaceVariant
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp)
        )
    }
}

fun versionTypeLabel(versionType: String): String = when (versionType) {
    "SOURCE" -> "来源解析版"
    "ADAPTED" -> "社区改编版"
    "KITCHEN_FIT" -> "个人适配版"
    else -> versionType
}

/** 秒数格式化为 mm:ss，便于对照原视频时间点；负值按 0 处理。 */
fun formatTimestamp(seconds: Int): String {
    val clamped = seconds.coerceAtLeast(0)
    return String.format(java.util.Locale.US, "%d:%02d", clamped / 60, clamped % 60)
}
