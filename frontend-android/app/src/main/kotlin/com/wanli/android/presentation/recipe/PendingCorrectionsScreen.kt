package com.wanli.android.presentation.recipe

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.TaskAlt
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateMapOf
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.data.remote.PendingCorrectionDto
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.WanLiTopBar

@Composable
fun PendingCorrectionsScreen(
    onBack: () -> Unit,
    viewModel: PendingCorrectionsViewModel = hiltViewModel()
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    val rejectionReasons = remember { mutableStateMapOf<String, String>() }
    Scaffold(
        topBar = { WanLiTopBar(title = "待审核纠错", onBack = onBack) }
    ) { padding ->
        Box(Modifier.fillMaxSize().padding(padding)) {
            when (val current = state) {
                PendingCorrectionsUiState.Loading -> Column(
                    modifier = Modifier.padding(20.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    repeat(4) { SkeletonCard() }
                }
                is PendingCorrectionsUiState.Error -> EmptyState(
                    icon = Icons.Outlined.ErrorOutline,
                    title = "加载失败",
                    subtitle = current.message,
                    modifier = Modifier.align(Alignment.Center),
                    primaryLabel = "重试",
                    onPrimary = viewModel::load
                )
                is PendingCorrectionsUiState.Success -> {
                    if (current.items.isEmpty()) {
                        EmptyState(
                            icon = Icons.Outlined.TaskAlt,
                            title = "暂无待审核建议",
                            subtitle = "所有纠错建议都已处理完毕",
                            modifier = Modifier.align(Alignment.Center)
                        )
                    } else {
                        LazyColumn(Modifier.fillMaxSize().padding(16.dp)) {
                            current.message?.let { message ->
                                item { Text(message, modifier = Modifier.padding(bottom = 8.dp)) }
                            }
                            items(current.items, key = { it.id }) { proposal ->
                                CorrectionCard(
                                    proposal = proposal,
                                    working = current.workingId == proposal.id,
                                    rejectionReason = rejectionReasons[proposal.id].orEmpty(),
                                    onReasonChange = { rejectionReasons[proposal.id] = it },
                                    onApprove = { viewModel.review(proposal.id, approve = true) },
                                    onReject = {
                                        viewModel.review(
                                            proposal.id,
                                            approve = false,
                                            rejectionReason = rejectionReasons[proposal.id].orEmpty()
                                        )
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

@Composable
private fun CorrectionCard(
    proposal: PendingCorrectionDto,
    working: Boolean,
    rejectionReason: String,
    onReasonChange: (String) -> Unit,
    onApprove: () -> Unit,
    onReject: () -> Unit
) {
    Card(
        elevation = CardDefaults.cardElevation(defaultElevation = 1.dp),
        modifier = Modifier.fillMaxWidth().padding(bottom = 12.dp)
    ) {
        Column(Modifier.padding(14.dp)) {
            Text(proposal.recipe_title, style = MaterialTheme.typography.titleMedium)
            Text(
                "提议人：${proposal.proposer_nickname.ifBlank { "匿名用户" }}",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )
            Text("理由：${proposal.reason}", modifier = Modifier.padding(top = 6.dp))
            Text(
                "建议内容：${proposal.suggested_changes.ingredients.size} 项食材，" +
                    "${proposal.suggested_changes.steps.size} 个步骤",
                style = MaterialTheme.typography.bodySmall,
                modifier = Modifier.padding(top = 4.dp)
            )
            if (proposal.suggested_changes.title != proposal.recipe_title) {
                Text(
                    "建议菜名：${proposal.suggested_changes.title}",
                    style = MaterialTheme.typography.bodySmall,
                    modifier = Modifier.padding(top = 4.dp)
                )
            }
            proposal.suggested_changes.ingredients.forEach { ingredient ->
                Text(
                    "· ${ingredient.name} ${ingredient.amount.orEmpty()}",
                    style = MaterialTheme.typography.bodySmall
                )
            }
            proposal.suggested_changes.steps.forEach { step ->
                Text(
                    "${step.step_order}. ${step.description}",
                    style = MaterialTheme.typography.bodySmall
                )
            }
            OutlinedTextField(
                value = rejectionReason,
                onValueChange = onReasonChange,
                label = { Text("拒绝理由（拒绝时必填）") },
                modifier = Modifier.fillMaxWidth().padding(top = 8.dp)
            )
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                modifier = Modifier.fillMaxWidth().padding(top = 10.dp)
            ) {
                OutlinedButton(onClick = onReject, enabled = !working, modifier = Modifier.weight(1f)) {
                    Text("拒绝")
                }
                Button(onClick = onApprove, enabled = !working, modifier = Modifier.weight(1f)) {
                    Text(if (working) "处理中…" else "通过")
                }
            }
        }
    }
}
