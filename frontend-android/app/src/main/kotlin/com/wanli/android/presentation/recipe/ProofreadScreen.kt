package com.wanli.android.presentation.recipe

import androidx.compose.foundation.background
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
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.DragIndicator
import androidx.compose.material.icons.outlined.Restaurant
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.ui.Alignment
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.wanli.android.presentation.components.ErrorMessage
import com.wanli.android.presentation.components.SectionHeader
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.SocialBadge
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.components.cardContentPadding
import com.wanli.android.presentation.theme.TomatoContainer
import com.wanli.android.presentation.theme.TomatoSoft

@Composable
fun ProofreadScreen(
    onBack: () -> Unit,
    onPublished: (versionId: String) -> Unit,
    viewModel: ProofreadViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    // 发布成功后跳转菜谱详情页；消费后回到编辑态，避免配置变化时重复导航
    LaunchedEffect(uiState) {
        val state = uiState
        if (state is ProofreadUiState.Published) {
            onPublished(state.versionId)
            viewModel.consumePublished()
        }
    }

    Scaffold(
        topBar = { WanLiTopBar(title = "校对菜谱", onBack = onBack, modifier = Modifier.statusBarsPadding()) },
        bottomBar = {
            val state = uiState as? ProofreadUiState.Editing
            if (state != null) {
                ProofreadBottomBar(
                    working = state.working,
                    onSave = { viewModel.save() },
                    onPublish = { viewModel.publish() }
                )
            }
        }
    ) { padding ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
        ) {
            when (val state = uiState) {
                is ProofreadUiState.Loading -> Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(16.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    SkeletonCard()
                    SkeletonCard()
                }
                is ProofreadUiState.Error -> Column(
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(24.dp)
                ) {
                    ErrorMessage(state.message, Modifier.weight(1f))
                    Button(
                        onClick = { viewModel.load() },
                        modifier = Modifier.align(Alignment.CenterHorizontally)
                    ) {
                        Text("重试")
                    }
                }
                is ProofreadUiState.Published -> CircularProgressIndicator(
                    modifier = Modifier.align(Alignment.Center)
                )
                is ProofreadUiState.Editing -> ProofreadFormContent(state, viewModel)
            }
        }
    }
}

/** 底部固定操作条：发布为主操作，保存草稿为次操作。 */
@Composable
private fun ProofreadBottomBar(
    working: Boolean,
    onSave: () -> Unit,
    onPublish: () -> Unit
) {
    Surface(shadowElevation = 8.dp, modifier = Modifier.navigationBarsPadding()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            OutlinedButton(
                onClick = onSave,
                enabled = !working,
                modifier = Modifier.weight(1f)
            ) {
                Text("保存草稿")
            }
            Button(
                onClick = onPublish,
                enabled = !working,
                modifier = Modifier.weight(1f)
            ) {
                Text("发布到社区")
            }
        }
    }
}

@Composable
private fun ProofreadFormContent(
    state: ProofreadUiState.Editing,
    viewModel: ProofreadViewModel
) {
    val form = state.form
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp)
    ) {
        Text(
            "AI 已生成草稿，请核对食材与步骤后发布",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        // 摘要卡：封面 + 菜名编辑 + 来源行
        SummaryCard(state = state, viewModel = viewModel)

        // 食材卡
        SectionHeader("食材")
        IngredientsCard(form = form, viewModel = viewModel)

        // 步骤卡
        SectionHeader("步骤")
        StepsCard(form = form, viewModel = viewModel)

        state.hint?.let {
            Text(
                it,
                style = MaterialTheme.typography.bodySmall,
                color = if (state.isError) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary,
                modifier = Modifier.padding(top = 12.dp, bottom = 8.dp)
            )
        }
    }
}

@Composable
private fun SummaryCard(
    state: ProofreadUiState.Editing,
    viewModel: ProofreadViewModel
) {
    val form = state.form
    WanLiCard(
        modifier = Modifier
            .fillMaxWidth()
            .padding(top = 12.dp)
    ) {
        Column(modifier = Modifier.cardContentPadding()) {
            Row(verticalAlignment = Alignment.CenterVertically) {
                if (!state.coverUrl.isNullOrBlank()) {
                    AsyncImage(
                        model = state.coverUrl,
                        contentDescription = form.title,
                        contentScale = ContentScale.Crop,
                        modifier = Modifier
                            .size(64.dp)
                            .clip(MaterialTheme.shapes.small)
                    )
                } else {
                    Box(
                        modifier = Modifier
                            .size(64.dp)
                            .clip(MaterialTheme.shapes.small)
                            .background(TomatoSoft),
                        contentAlignment = Alignment.Center
                    ) {
                        Icon(
                            imageVector = Icons.Outlined.Restaurant,
                            contentDescription = null,
                            tint = MaterialTheme.colorScheme.primary
                        )
                    }
                }
                Column(modifier = Modifier.padding(start = 12.dp)) {
                    OutlinedTextField(
                        value = form.title,
                        onValueChange = { title -> viewModel.updateForm { it.copy(title = title) } },
                        label = { Text("菜名") },
                        singleLine = true,
                        modifier = Modifier.fillMaxWidth()
                    )
                    Text(
                        text = if (state.upNickname.isNotBlank()) "来源：B 站 · ${state.upNickname}" else "来源：B 站",
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        modifier = Modifier.padding(top = 4.dp)
                    )
                }
            }

            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                OutlinedTextField(
                    value = form.servings,
                    onValueChange = { servings -> viewModel.updateForm { it.copy(servings = servings) } },
                    label = { Text("分量") },
                    singleLine = true,
                    modifier = Modifier
                        .weight(1f)
                        .padding(top = 8.dp)
                )
                OutlinedTextField(
                    value = form.estimatedMinutes,
                    onValueChange = { minutes -> viewModel.updateForm { it.copy(estimatedMinutes = minutes) } },
                    label = { Text("耗时(分钟)") },
                    singleLine = true,
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                    modifier = Modifier
                        .weight(1f)
                        .padding(top = 8.dp)
                )
            }

            OutlinedTextField(
                value = form.instructions,
                onValueChange = { instructions -> viewModel.updateForm { it.copy(instructions = instructions) } },
                label = { Text("小贴士（可选）") },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 8.dp)
            )
        }
    }
}

@Composable
private fun IngredientsCard(
    form: ProofreadForm,
    viewModel: ProofreadViewModel
) {
    WanLiCard(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.cardContentPadding()) {
            form.ingredients.forEachIndexed { index, ingredient ->
                key(index) {
                    Column(modifier = Modifier.padding(top = 4.dp)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Icon(
                                imageVector = Icons.Default.DragIndicator,
                                contentDescription = null,
                                tint = MaterialTheme.colorScheme.outline,
                                modifier = Modifier.size(20.dp)
                            )
                            OutlinedTextField(
                                value = ingredient.name,
                                onValueChange = { viewModel.updateIngredient(index, ingredient.copy(name = it)) },
                                placeholder = { Text("食材名") },
                                singleLine = true,
                                modifier = Modifier
                                    .weight(1f)
                                    .padding(start = 4.dp)
                            )
                            IconButton(onClick = { viewModel.removeIngredient(index) }) {
                                Icon(
                                    Icons.Default.Close,
                                    contentDescription = "删除食材 ${ingredient.name.ifBlank { "第 ${index + 1} 项" }}"
                                )
                            }
                        }
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            OutlinedTextField(
                                value = ingredient.amount,
                                onValueChange = { viewModel.updateIngredient(index, ingredient.copy(amount = it)) },
                                placeholder = { Text("用量") },
                                singleLine = true,
                                modifier = Modifier
                                    .weight(1f)
                                    .padding(start = 24.dp, top = 4.dp)
                            )
                            TextButton(
                                onClick = {
                                    val next = CATEGORY_ORDER[(CATEGORY_ORDER.indexOf(ingredient.category) + 1) % CATEGORY_ORDER.size]
                                    viewModel.updateIngredient(index, ingredient.copy(category = next))
                                }
                            ) {
                                Text(categoryLabel(ingredient.category), style = MaterialTheme.typography.bodySmall)
                            }
                            if (!ingredient.isExplicit) {
                                SocialBadge("AI 推测")
                            }
                        }
                    }
                }
            }
            OutlinedButton(
                onClick = { viewModel.addIngredient() },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 8.dp)
            ) {
                Text("+ 添加食材")
            }
        }
    }
}

@Composable
private fun StepsCard(
    form: ProofreadForm,
    viewModel: ProofreadViewModel
) {
    WanLiCard(modifier = Modifier.fillMaxWidth()) {
        Column(modifier = Modifier.cardContentPadding()) {
            form.steps.forEachIndexed { index, step ->
                key(index) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.padding(top = 4.dp)
                    ) {
                        Box(
                            modifier = Modifier
                                .size(24.dp)
                                .background(TomatoContainer, CircleShape),
                            contentAlignment = Alignment.Center
                        ) {
                            Text(
                                "${index + 1}",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.primary
                            )
                        }
                        OutlinedTextField(
                            value = step.description,
                            onValueChange = { viewModel.updateStep(index, step.copy(description = it)) },
                            placeholder = { Text("步骤描述") },
                            modifier = Modifier
                                .weight(1f)
                                .padding(start = 8.dp)
                        )
                        step.videoTimestamp.toIntOrNull()?.let { seconds ->
                            TimestampChip(seconds = seconds)
                        }
                        IconButton(onClick = { viewModel.removeStep(index) }) {
                            Icon(Icons.Default.Close, contentDescription = "删除第 ${index + 1} 步")
                        }
                    }
                }
            }
            OutlinedButton(
                onClick = { viewModel.addStep() },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(top = 8.dp)
            ) {
                Text("+ 添加步骤")
            }
        }
    }
}

/** 步骤时间戳 chip：显示视频中的 mm:ss 位置。 */
@Composable
private fun TimestampChip(seconds: Int) {
    Surface(
        shape = MaterialTheme.shapes.small,
        color = MaterialTheme.colorScheme.primaryContainer,
        modifier = Modifier.padding(start = 4.dp)
    ) {
        Text(
            text = formatTimestamp(seconds),
            style = MaterialTheme.typography.labelSmall,
            color = MaterialTheme.colorScheme.onPrimaryContainer,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
        )
    }
}
