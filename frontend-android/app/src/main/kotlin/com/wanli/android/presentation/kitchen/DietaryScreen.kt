package com.wanli.android.presentation.kitchen

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.BackHand
import androidx.compose.material.icons.outlined.Block
import androidx.compose.material.icons.outlined.Close
import androidx.compose.material.icons.outlined.MedicalServices
import androidx.compose.material3.Button
import androidx.compose.material3.FilterChip
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.data.remote.RestrictionDto
import com.wanli.android.presentation.components.ErrorMessage
import com.wanli.android.presentation.components.SectionHeader
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.StatusBanner
import com.wanli.android.presentation.components.StatusBannerType
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiFilterChip
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding

private val RESTRICTION_TYPES = listOf("ALLERGY" to "过敏", "DISLIKE" to "不吃", "MEDICAL" to "医疗忌口")

@Composable
fun DietaryScreen(
    onBack: () -> Unit,
    viewModel: DietaryViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = { WanLiTopBar(title = "忌口管理", onBack = onBack) }
    ) { padding ->
        WanLiPullToRefreshBox(
            isRefreshing = uiState is DietaryUiState.Loading,
            onRefresh = { viewModel.loadRestrictions() },
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
                is DietaryUiState.Loading -> {
                    SkeletonCard(modifier = Modifier.padding(top = 16.dp))
                    SkeletonCard(modifier = Modifier.padding(top = 16.dp))
                }
                is DietaryUiState.Error -> {
                    ErrorMessage(message = state.message)
                    Button(
                        onClick = { viewModel.loadRestrictions() },
                        modifier = Modifier.padding(top = 16.dp)
                    ) {
                        Text("重试")
                    }
                }
                is DietaryUiState.Success -> DietaryContent(
                    restrictions = state.restrictions,
                    onSave = { viewModel.saveRestrictions(it) }
                )
            }
        }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun DietaryContent(
    restrictions: List<RestrictionDto>,
    onSave: (List<RestrictionDto>) -> Unit
) {
    var newIngredient by remember { mutableStateOf("") }
    var selectedType by remember { mutableStateOf("ALLERGY") }

    // 过敏具有最高拦截优先级，顶部常驻提示
    StatusBanner(type = StatusBannerType.WARNING) {
        Text(
            text = "过敏与医疗忌口在菜谱适配中会被强制拦截，请如实填写",
            style = MaterialTheme.typography.bodySmall
        )
    }

    WanLiCard(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 16.dp)
    ) {
        Column(modifier = Modifier.cardContentPadding()) {
            SectionHeader(title = "我的忌口")

            if (restrictions.isEmpty()) {
                Text(
                    text = "还没有忌口项，在下方添加",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    modifier = Modifier.padding(top = 8.dp)
                )
            } else {
                FlowRow(modifier = Modifier.fillMaxWidth().padding(top = 8.dp)) {
                    restrictions.forEach { restriction ->
                        RestrictionTag(
                            restriction = restriction,
                            onRemove = {
                                onSave(restrictions - restriction)
                            },
                            modifier = Modifier.padding(end = 8.dp, bottom = 8.dp)
                        )
                    }
                }
            }
        }
    }

    WanLiCard(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 16.dp)
    ) {
        Column(modifier = Modifier.cardContentPadding()) {
            SectionHeader(title = "添加忌口")

            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.padding(top = 8.dp)
            ) {
                RESTRICTION_TYPES.forEach { (type, label) ->
                    WanLiFilterChip(
                        selected = selectedType == type,
                        onClick = { selectedType = type },
                        label = label,
                    )
                }
            }

            Row(
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 8.dp)
            ) {
                OutlinedTextField(
                    value = newIngredient,
                    onValueChange = { newIngredient = it },
                    label = { Text("食材/品类名") },
                    singleLine = true,
                    shape = RoundedCornerShape(16.dp),
                    modifier = Modifier.weight(1f)
                )
                Button(
                    onClick = {
                        if (newIngredient.isNotBlank()) {
                            val updated = restrictions + RestrictionDto(
                                restriction_type = selectedType,
                                ingredient_name = newIngredient
                            )
                            onSave(updated)
                            newIngredient = ""
                        }
                    },
                    enabled = newIngredient.isNotBlank()
                ) {
                    Text("添加")
                }
            }
        }
    }
}

@Composable
private fun RestrictionTag(
    restriction: RestrictionDto,
    onRemove: () -> Unit,
    modifier: Modifier = Modifier
) {
    val (icon, label, containerColor, contentColor) = when (restriction.restriction_type) {
        "ALLERGY" -> TagStyle(
            Icons.Outlined.Block, "过敏",
            MaterialTheme.colorScheme.errorContainer,
            MaterialTheme.colorScheme.onErrorContainer
        )
        "MEDICAL" -> TagStyle(
            Icons.Outlined.MedicalServices, "医疗忌口",
            MaterialTheme.colorScheme.tertiaryContainer,
            MaterialTheme.colorScheme.onTertiaryContainer
        )
        else -> TagStyle(
            Icons.Outlined.BackHand, "不吃",
            MaterialTheme.colorScheme.surfaceVariant,
            MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
    Surface(
        color = containerColor,
        shape = RoundedCornerShape(12.dp),
        modifier = modifier
    ) {
        Row(
            verticalAlignment = Alignment.CenterVertically,
            modifier = Modifier.padding(start = 12.dp, top = 4.dp, bottom = 4.dp, end = 4.dp)
        ) {
            Icon(
                imageVector = icon,
                contentDescription = null,
                tint = contentColor,
                modifier = Modifier
                    .size(16.dp)
                    .padding(end = 4.dp)
            )
            Text(
                text = "${restriction.ingredient_name} · $label",
                style = MaterialTheme.typography.bodyMedium,
                color = contentColor
            )
            IconButton(
                onClick = onRemove,
                modifier = Modifier.size(32.dp)
            ) {
                Icon(
                    imageVector = Icons.Outlined.Close,
                    contentDescription = "删除${restriction.ingredient_name}",
                    tint = contentColor.copy(alpha = 0.7f),
                    modifier = Modifier.size(16.dp)
                )
            }
        }
    }
}

private data class TagStyle(
    val icon: ImageVector,
    val label: String,
    val containerColor: Color,
    val contentColor: Color
)
