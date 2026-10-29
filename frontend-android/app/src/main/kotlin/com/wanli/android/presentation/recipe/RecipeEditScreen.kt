package com.wanli.android.presentation.recipe

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Close
import androidx.compose.material3.Button
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.presentation.components.WanLiTopBar

@Composable
fun RecipeEditScreen(
    onBack: () -> Unit,
    onSubmitted: (versionId: String?) -> Unit,
    viewModel: RecipeEditViewModel = hiltViewModel()
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val isCorrection = viewModel.mode == EDIT_MODE_CORRECTION

    LaunchedEffect(state) {
        (state as? RecipeEditUiState.Submitted)?.let { onSubmitted(it.versionId) }
    }

    Scaffold(
        topBar = {
            WanLiTopBar(
                title = if (isCorrection) "来源纠错" else "派生改编",
                onBack = onBack,
                modifier = Modifier.statusBarsPadding(),
            )
        }
    ) { padding ->
        Box(Modifier.fillMaxSize().padding(padding)) {
            when (val current = state) {
                RecipeEditUiState.Loading, is RecipeEditUiState.Submitted ->
                    CircularProgressIndicator(Modifier.align(Alignment.Center))
                is RecipeEditUiState.Error -> Column(
                    modifier = Modifier.align(Alignment.Center),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Text(current.message, color = MaterialTheme.colorScheme.error)
                    TextButton(onClick = viewModel::load) { Text("重试") }
                }
                is RecipeEditUiState.Editing -> EditForm(
                    state = current,
                    isCorrection = isCorrection,
                    viewModel = viewModel
                )
            }
        }
    }
}

@Composable
private fun EditForm(
    state: RecipeEditUiState.Editing,
    isCorrection: Boolean,
    viewModel: RecipeEditViewModel
) {
    val form = state.form
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp)
    ) {
        Text(
            if (isCorrection) "只修正与来源视频不一致的内容，提交后由来源发布者审核"
            else "改编会立即作为公开版本发布，并保留当前版本为父节点",
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        OutlinedTextField(
            value = form.title,
            onValueChange = { value -> viewModel.updateForm { it.copy(title = value) } },
            label = { Text("版本菜名") },
            modifier = Modifier.fillMaxWidth().padding(top = 12.dp)
        )
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
            OutlinedTextField(
                value = form.servings,
                onValueChange = { value -> viewModel.updateForm { it.copy(servings = value) } },
                label = { Text("分量") },
                modifier = Modifier.weight(1f)
            )
            OutlinedTextField(
                value = form.estimatedMinutes,
                onValueChange = { value -> viewModel.updateForm { it.copy(estimatedMinutes = value) } },
                label = { Text("耗时(分钟)") },
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                modifier = Modifier.weight(1f)
            )
        }
        OutlinedTextField(
            value = form.instructions,
            onValueChange = { value -> viewModel.updateForm { it.copy(instructions = value) } },
            label = { Text("提示与注意事项") },
            modifier = Modifier.fillMaxWidth().padding(top = 8.dp)
        )

        EditSectionHeader("食材", viewModel::addIngredient)
        form.ingredients.forEachIndexed { index, ingredient ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                OutlinedTextField(
                    value = ingredient.name,
                    onValueChange = { viewModel.updateIngredient(index, ingredient.copy(name = it)) },
                    label = { Text("食材") },
                    modifier = Modifier.weight(1.5f)
                )
                OutlinedTextField(
                    value = ingredient.amount,
                    onValueChange = { viewModel.updateIngredient(index, ingredient.copy(amount = it)) },
                    label = { Text("用量") },
                    modifier = Modifier.weight(1f).padding(start = 4.dp)
                )
                TextButton(onClick = {
                    val categoryIndex = CATEGORY_ORDER.indexOf(ingredient.category)
                    val next = CATEGORY_ORDER[(categoryIndex + 1) % CATEGORY_ORDER.size]
                    viewModel.updateIngredient(index, ingredient.copy(category = next))
                }) {
                    Text(categoryLabel(ingredient.category))
                }
                IconButton(onClick = { viewModel.removeIngredient(index) }) {
                    Icon(Icons.Default.Close, contentDescription = "删除食材")
                }
            }
        }

        EditSectionHeader("步骤", viewModel::addStep)
        form.steps.forEachIndexed { index, step ->
            Row(verticalAlignment = Alignment.CenterVertically) {
                Text("${index + 1}.", color = MaterialTheme.colorScheme.primary)
                OutlinedTextField(
                    value = step.description,
                    onValueChange = { viewModel.updateStep(index, step.copy(description = it)) },
                    label = { Text("步骤描述") },
                    modifier = Modifier.weight(1f).padding(start = 4.dp)
                )
                IconButton(onClick = { viewModel.removeStep(index) }) {
                    Icon(Icons.Default.Close, contentDescription = "删除步骤")
                }
            }
        }

        OutlinedTextField(
            value = state.note,
            onValueChange = viewModel::updateNote,
            label = { Text(if (isCorrection) "纠错理由" else "改编说明") },
            minLines = 2,
            modifier = Modifier.fillMaxWidth().padding(top = 16.dp)
        )
        state.hint?.let {
            Text(it, color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(top = 8.dp))
        }
        Button(
            onClick = viewModel::submit,
            enabled = !state.working,
            modifier = Modifier.fillMaxWidth().padding(vertical = 16.dp)
        ) {
            Text(
                if (state.working) "提交中…"
                else if (isCorrection) "提交纠错建议"
                else "发布改编版"
            )
        }
    }
}

@Composable
private fun EditSectionHeader(title: String, onAdd: () -> Unit) {
    Row(
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier.fillMaxWidth().padding(top = 16.dp)
    ) {
        Text(title, style = MaterialTheme.typography.titleMedium)
        TextButton(onClick = onAdd) {
            Icon(Icons.Default.Add, contentDescription = null)
            Text("添加")
        }
    }
}
