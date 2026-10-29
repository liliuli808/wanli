package com.wanli.android.presentation.recipe

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.outlined.DinnerDining
import androidx.compose.material.icons.outlined.Lightbulb
import androidx.compose.material.icons.outlined.SwapHoriz
import androidx.compose.material.icons.outlined.WarningAmber
import androidx.compose.material3.Button
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
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextDecoration
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.data.remote.AdaptationNotesDto
import com.wanli.android.data.remote.AdaptationSubstitutionDto
import com.wanli.android.presentation.components.ErrorMessage
import com.wanli.android.presentation.components.SectionHeader
import com.wanli.android.presentation.components.StatusBanner
import com.wanli.android.presentation.components.StatusBannerType
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding
import com.wanli.android.presentation.theme.CaramelDeep
import com.wanli.android.presentation.theme.CaramelSoft
import com.wanli.android.presentation.theme.WarningAmber

@Composable
fun AdaptRecipeScreen(
    onBack: () -> Unit,
    onOpenResult: (String) -> Unit,
    viewModel: AdaptRecipeViewModel = hiltViewModel()
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val successState = state as? AdaptRecipeUiState.Success
    Scaffold(
        topBar = { WanLiTopBar(title = "适配我的厨房", onBack = onBack) },
        bottomBar = {
            if (successState != null) {
                AdaptResultBottomBar(
                    onDone = { onOpenResult(successState.detail.id) },
                    onViewOriginal = onBack
                )
            }
        }
    ) { padding ->
        Box(Modifier.fillMaxSize().padding(padding)) {
            when (val current = state) {
                AdaptRecipeUiState.Ready -> Column(
                    modifier = Modifier.align(Alignment.Center).padding(24.dp),
                    horizontalAlignment = Alignment.CenterHorizontally,
                    verticalArrangement = Arrangement.spacedBy(16.dp)
                ) {
                    Icon(
                        imageVector = Icons.Outlined.DinnerDining,
                        contentDescription = null,
                        tint = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.size(64.dp)
                    )
                    Text("将根据“我的厨房”和忌口生成一个默认私有的个人适配版。")
                    Text(
                        "过敏源与医疗忌口会优先进行安全校验。",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Button(onClick = viewModel::adapt) { Text("开始适配") }
                }
                AdaptRecipeUiState.Loading -> Column(
                    modifier = Modifier.align(Alignment.Center),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    CircularProgressIndicator()
                    Text("正在检查厨房条件并生成适配版…", modifier = Modifier.padding(top = 12.dp))
                }
                is AdaptRecipeUiState.Error -> Column(
                    modifier = Modifier.fillMaxSize().padding(24.dp)
                ) {
                    ErrorMessage(current.message, Modifier.weight(1f))
                    Button(
                        onClick = viewModel::adapt,
                        modifier = Modifier.align(Alignment.CenterHorizontally)
                    ) { Text("重试") }
                }
                is AdaptRecipeUiState.Success -> AdaptationResultContent(
                    title = current.detail.title,
                    notes = current.detail.adaptation_notes
                )
            }
        }
    }
}

@Composable
private fun AdaptResultBottomBar(onDone: () -> Unit, onViewOriginal: () -> Unit) {
    Surface(shadowElevation = 8.dp, modifier = Modifier.navigationBarsPadding()) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            OutlinedButton(onClick = onViewOriginal, modifier = Modifier.weight(1f)) {
                Text("查看原步骤")
            }
            Button(onClick = onDone, modifier = Modifier.weight(1f)) {
                Text("完成")
            }
        }
    }
}

@Composable
private fun AdaptationResultContent(
    title: String,
    notes: AdaptationNotesDto?
) {
    Column(
        modifier = Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        AdaptSuccessBanner(notes)
        Text(title, style = MaterialTheme.typography.titleLarge)
        Text(
            "已私有保存，可在“我的适配版”中选择公开分享。",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )
        notes?.let {
            SubstitutionSection(it.substitutions)
            MissingSection(it.missing_ingredients + it.missing_tools)
            FlavorSection(it.flavor_changes)
        }
    }
}

@Composable
private fun AdaptSuccessBanner(notes: AdaptationNotesDto?) {
    StatusBanner(type = StatusBannerType.SUCCESS) {
        Text(
            "已根据你的厨房完成适配",
            style = MaterialTheme.typography.titleMedium,
            fontWeight = FontWeight.SemiBold,
            // 多行文案使用深色文字以保证可读性，不沿用语义绿色
            color = MaterialTheme.colorScheme.onSurface
        )
        if (notes != null) {
            val substitutionCount = notes.substitutions.size
            val missingCount = notes.missing_ingredients.size + notes.missing_tools.size
            val flavorCount = notes.flavor_changes.size
            Text(
                if (substitutionCount == 0 && missingCount == 0 && flavorCount == 0) {
                    "无需替换或补充，可直接烹饪"
                } else {
                    "$substitutionCount 处替换 · $missingCount 种缺少 · $flavorCount 条口味建议"
                },
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
        }
    }
}

@Composable
private fun SubstitutionSection(substitutions: List<AdaptationSubstitutionDto>) {
    if (substitutions.isEmpty()) return
    SectionHeader("替换")
    substitutions.forEach { item ->
        WanLiCard(Modifier.fillMaxWidth()) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .cardContentPadding()
                    .semantics(mergeDescendants = true) {
                        contentDescription = "${item.original}，已替换为 ${item.replacement}。原因：${item.reason}"
                    },
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Outlined.SwapHoriz,
                    contentDescription = null,
                    tint = MaterialTheme.colorScheme.primary
                )
                Column(Modifier.padding(start = 12.dp)) {
                    Text(
                        item.original,
                        style = MaterialTheme.typography.bodyMedium.copy(
                            textDecoration = TextDecoration.LineThrough
                        ),
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.padding(top = 2.dp)
                    ) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowForward,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.onSurfaceVariant,
                            modifier = Modifier
                                .padding(end = 6.dp)
                                .size(16.dp)
                        )
                        Text(
                            item.replacement,
                            style = MaterialTheme.typography.bodyMedium,
                            color = MaterialTheme.colorScheme.primary,
                            fontWeight = FontWeight.SemiBold
                        )
                    }
                    Text(
                        item.reason,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 4.dp)
                    )
                }
            }
        }
    }
}

@Composable
private fun MissingSection(missing: List<String>) {
    if (missing.isEmpty()) return
    SectionHeader("缺少")
    missing.forEach { name ->
        WanLiCard(Modifier.fillMaxWidth()) {
            Row(
                modifier = Modifier.fillMaxWidth().cardContentPadding(),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Icon(
                    imageVector = Icons.Outlined.WarningAmber,
                    contentDescription = null,
                    tint = WarningAmber
                )
                Text(
                    name,
                    style = MaterialTheme.typography.bodyMedium,
                    modifier = Modifier.weight(1f).padding(start = 12.dp)
                )
                Text(
                    "可省略",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        }
    }
}

@Composable
private fun FlavorSection(flavorChanges: List<String>) {
    if (flavorChanges.isEmpty()) return
    SectionHeader("口味变化")
    Surface(
        color = CaramelSoft,
        shape = MaterialTheme.shapes.medium,
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier.padding(16.dp),
            verticalAlignment = Alignment.Top
        ) {
            Icon(
                imageVector = Icons.Outlined.Lightbulb,
                contentDescription = null,
                tint = CaramelDeep
            )
            Column(
                modifier = Modifier.padding(start = 12.dp),
                verticalArrangement = Arrangement.spacedBy(4.dp)
            ) {
                flavorChanges.forEach {
                    Text(it, style = MaterialTheme.typography.bodyMedium)
                }
            }
        }
    }
}

@Composable
internal fun AdaptationListCard(title: String, items: List<String>) {
    if (items.isEmpty()) return
    Card(
        elevation = CardDefaults.cardElevation(defaultElevation = 1.dp),
        modifier = Modifier.fillMaxWidth()
    ) {
        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
            Text(title, style = MaterialTheme.typography.titleSmall)
            items.forEach { Text("• $it", style = MaterialTheme.typography.bodyMedium) }
        }
    }
}
