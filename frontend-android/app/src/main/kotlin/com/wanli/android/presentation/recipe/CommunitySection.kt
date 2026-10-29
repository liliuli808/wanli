package com.wanli.android.presentation.recipe

import android.net.Uri
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.PickVisualMediaRequest
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.StarOutline
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.wanli.android.presentation.components.AuthorRow
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.RatingStars
import com.wanli.android.presentation.components.SectionHeader
import com.wanli.android.presentation.theme.CaramelAccent

@Composable
fun CommunitySection(
    viewModel: CommunityViewModel = hiltViewModel()
) {
    val state by viewModel.uiState.collectAsStateWithLifecycle()
    var comment by rememberSaveable { mutableStateOf("") }
    var showCookingDialog by rememberSaveable { mutableStateOf(false) }

    Column(
        modifier = Modifier.fillMaxWidth().padding(top = 20.dp, bottom = 24.dp),
        verticalArrangement = Arrangement.spacedBy(10.dp)
    ) {
        when (val current = state) {
            CommunityUiState.Loading -> Box(
                modifier = Modifier.fillMaxWidth(),
                contentAlignment = Alignment.Center
            ) {
                CircularProgressIndicator()
            }
            is CommunityUiState.Error -> EmptyState(
                icon = Icons.Outlined.ErrorOutline,
                title = "加载失败",
                subtitle = current.message,
                primaryLabel = "重试",
                onPrimary = viewModel::load
            )
            is CommunityUiState.Success -> {
                current.message?.let {
                    Text(it, color = MaterialTheme.colorScheme.error)
                }
                Button(
                    onClick = { showCookingDialog = true },
                    enabled = !current.submitting,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text("我做过")
                }

                // 做过墙：有成品图的记录横排展示
                val photoRecords = current.records.filter { !it.photo_url.isNullOrBlank() }
                if (photoRecords.isNotEmpty()) {
                    SectionHeader("做过墙")
                    LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        items(photoRecords, key = { it.id }) { record ->
                            AsyncImage(
                                model = record.photo_url,
                                contentDescription = "${record.author_nickname} 的成品图",
                                modifier = Modifier
                                    .size(96.dp)
                                    .clip(MaterialTheme.shapes.small),
                                contentScale = ContentScale.Crop,
                            )
                        }
                    }
                }

                current.records.forEach { record ->
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            Modifier.padding(12.dp),
                            verticalArrangement = Arrangement.spacedBy(6.dp)
                        ) {
                            AuthorRow(
                                nickname = record.author_nickname.ifBlank { "碗里用户" },
                                subtitle = record.created_at?.take(10),
                                trailing = if (record.is_mine) {
                                    {
                                        TextButton(
                                            onClick = { viewModel.deleteCookingRecord(record.id) },
                                            enabled = !current.submitting
                                        ) { Text("删除") }
                                    }
                                } else null
                            )
                            RatingStars(record.rating)
                            record.feedback?.let {
                                Text(it, style = MaterialTheme.typography.bodyMedium)
                            }
                        }
                    }
                }

                SectionHeader("评论（${current.comments.size}）")
                OutlinedTextField(
                    value = comment,
                    onValueChange = { comment = it },
                    label = { Text("分享你的做法或建议") },
                    minLines = 2,
                    modifier = Modifier.fillMaxWidth()
                )
                Button(
                    onClick = {
                        viewModel.submitComment(comment)
                        comment = ""
                    },
                    enabled = comment.isNotBlank() && !current.submitting,
                    modifier = Modifier.fillMaxWidth()
                ) { Text("发表评论") }
                current.comments.forEach { item ->
                    Card(modifier = Modifier.fillMaxWidth()) {
                        Column(
                            Modifier.padding(12.dp),
                            verticalArrangement = Arrangement.spacedBy(6.dp)
                        ) {
                            AuthorRow(
                                nickname = item.author_nickname.ifBlank { "碗里用户" },
                                subtitle = item.created_at?.take(10),
                                trailing = if (item.is_mine) {
                                    {
                                        TextButton(
                                            onClick = { viewModel.deleteComment(item.id) },
                                            enabled = !current.submitting
                                        ) { Text("删除") }
                                    }
                                } else null
                            )
                            Text(item.content, style = MaterialTheme.typography.bodyMedium)
                        }
                    }
                }
            }
        }
    }

    if (showCookingDialog) {
        CookingRecordDialog(
            onDismiss = { showCookingDialog = false },
            onSubmit = { rating, feedback, photo ->
                viewModel.submitCookingRecord(rating, feedback, photo)
                showCookingDialog = false
            }
        )
    }
}

@Composable
private fun CookingRecordDialog(
    onDismiss: () -> Unit,
    onSubmit: (Int?, String, Uri?) -> Unit
) {
    var rating by rememberSaveable { mutableIntStateOf(0) }
    var feedback by rememberSaveable { mutableStateOf("") }
    var photoUri by rememberSaveable { mutableStateOf<Uri?>(null) }
    val picker = rememberLauncherForActivityResult(
        ActivityResultContracts.PickVisualMedia()
    ) { uri ->
        photoUri = uri
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("记录这次下厨") },
        text = {
            Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text("评分")
                Row {
                    (1..5).forEach { value ->
                        IconButton(onClick = { rating = value }) {
                            Icon(
                                imageVector = if (value <= rating) Icons.Filled.Star
                                    else Icons.Outlined.StarOutline,
                                contentDescription = "$value 星",
                                tint = CaramelAccent
                            )
                        }
                    }
                }
                OutlinedTextField(
                    value = feedback,
                    onValueChange = { feedback = it },
                    label = { Text("制作心得（可选）") },
                    minLines = 2,
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedButton(
                    onClick = {
                        picker.launch(
                            PickVisualMediaRequest(ActivityResultContracts.PickVisualMedia.ImageOnly)
                        )
                    }
                ) {
                    Text(if (photoUri == null) "选择成品图" else "已选择图片，点击更换")
                }
            }
        },
        confirmButton = {
            TextButton(
                onClick = {
                    onSubmit(rating.takeIf { it > 0 }, feedback, photoUri)
                },
                enabled = rating > 0 || feedback.isNotBlank() || photoUri != null
            ) { Text("提交") }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("取消") }
        }
    )
}
