package com.wanli.android.presentation.profile

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.outlined.Edit
import androidx.compose.material.icons.outlined.ErrorOutline
import androidx.compose.material.icons.outlined.PersonOutline
import androidx.compose.material.icons.outlined.RateReview
import androidx.compose.material.icons.outlined.Restaurant
import androidx.compose.material.icons.outlined.Shield
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import coil.compose.AsyncImage
import com.wanli.android.presentation.components.EmptyState
import com.wanli.android.presentation.components.ScrollableFill
import com.wanli.android.presentation.components.SectionHeader
import com.wanli.android.presentation.components.SkeletonCard
import com.wanli.android.presentation.components.UserAvatar
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.WanLiPullToRefreshBox
import com.wanli.android.presentation.components.WanLiTopBar

@Composable
fun ProfileScreen(
    isLoggedIn: Boolean,
    onLogout: () -> Unit,
    onLogin: () -> Unit,
    onOpenPendingCorrections: () -> Unit = {},
    onOpenMyAdaptations: () -> Unit = {},
    onOpenDietary: () -> Unit = {},
) {
    Scaffold(
        topBar = { WanLiTopBar(title = "我的") }
    ) { padding ->
        if (!isLoggedIn) {
            EmptyState(
                icon = Icons.Outlined.PersonOutline,
                title = "还未登录",
                subtitle = "登录后可使用厨房适配、纠错审核等功能",
                primaryLabel = "登录",
                onPrimary = onLogin,
                modifier = Modifier.padding(padding),
            )
            return@Scaffold
        }

        LoggedInProfile(
            onLogout = onLogout,
            onOpenPendingCorrections = onOpenPendingCorrections,
            onOpenMyAdaptations = onOpenMyAdaptations,
            onOpenDietary = onOpenDietary,
            padding = padding,
        )
    }
}

@Composable
private fun LoggedInProfile(
    onLogout: () -> Unit,
    onOpenPendingCorrections: () -> Unit,
    onOpenMyAdaptations: () -> Unit,
    onOpenDietary: () -> Unit,
    padding: PaddingValues,
    viewModel: ProfileViewModel = hiltViewModel(),
) {
        val uiState by viewModel.uiState.collectAsStateWithLifecycle()
        WanLiPullToRefreshBox(
            isRefreshing = uiState is ProfileUiState.Loading,
            onRefresh = viewModel::load,
            modifier = Modifier.fillMaxSize().padding(padding),
        ) {
        when (val state = uiState) {
            is ProfileUiState.Loading -> LazyColumn(
                contentPadding = PaddingValues(20.dp),
                verticalArrangement = Arrangement.spacedBy(16.dp),
                modifier = Modifier.fillMaxSize(),
            ) {
                items(3) { SkeletonCard() }
            }

            is ProfileUiState.Error -> ScrollableFill {
                EmptyState(
                    icon = Icons.Outlined.ErrorOutline,
                    title = "加载失败",
                    subtitle = state.message,
                    primaryLabel = "重试",
                    onPrimary = viewModel::load,
                )
            }

            is ProfileUiState.Success -> ProfileContent(
                state = state,
                onLogout = onLogout,
                onOpenPendingCorrections = onOpenPendingCorrections,
                onOpenMyAdaptations = onOpenMyAdaptations,
                onOpenDietary = onOpenDietary,
                onUpdateNickname = viewModel::updateNickname,
            )
        }
        }
}

@Composable
private fun ProfileContent(
    state: ProfileUiState.Success,
    onLogout: () -> Unit,
    onOpenPendingCorrections: () -> Unit,
    onOpenMyAdaptations: () -> Unit,
    onOpenDietary: () -> Unit,
    onUpdateNickname: (String) -> Unit,
    modifier: Modifier = Modifier,
) {
    var showEditDialog by rememberSaveable { mutableStateOf(false) }

    if (showEditDialog) {
        EditNicknameDialog(
            currentNickname = state.nickname,
            onDismiss = { showEditDialog = false },
            onConfirm = { nickname ->
                onUpdateNickname(nickname)
                showEditDialog = false
            },
        )
    }

    val photoRecords = remember(state.cookingRecords) {
        state.cookingRecords.filter { it.photoUrl != null || it.coverUrl != null }
    }

    LazyColumn(
        contentPadding = PaddingValues(20.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
        modifier = modifier.fillMaxSize(),
    ) {
        item {
            // 身份区：大头像 + 衬线昵称 + 邮箱 + 编辑入口
            Row(verticalAlignment = Alignment.CenterVertically) {
                UserAvatar(nickname = state.nickname, avatarUrl = state.avatarUrl, size = 64.dp)
                Spacer(Modifier.width(16.dp))
                Column(Modifier.weight(1f)) {
                    Text(state.nickname.ifBlank { "碗里的厨子" }, style = MaterialTheme.typography.headlineSmall)
                    Text(
                        state.email,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
                IconButton(onClick = { showEditDialog = true }) {
                    Icon(
                        Icons.Outlined.Edit,
                        contentDescription = "编辑资料",
                        tint = MaterialTheme.colorScheme.onSurfaceVariant,
                    )
                }
            }
        }

        if (photoRecords.isNotEmpty()) {
            item { SectionHeader("我的做过") }
            items(photoRecords.chunked(3), key = { row -> row.first().id }) { rowItems ->
                Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                    rowItems.forEach { record ->
                        AsyncImage(
                            model = record.photoUrl ?: record.coverUrl,
                            contentDescription = record.recipeTitle,
                            modifier = Modifier
                                .weight(1f)
                                .aspectRatio(1f)
                                .clip(MaterialTheme.shapes.small),
                            contentScale = ContentScale.Crop,
                        )
                    }
                    repeat(3 - rowItems.size) { Spacer(Modifier.weight(1f)) }
                }
            }
        }

        item {
            WanLiCard {
                MenuRow(icon = Icons.Outlined.Restaurant, title = "我的适配版", onClick = onOpenMyAdaptations)
                MenuRow(icon = Icons.Outlined.RateReview, title = "待审核纠错", onClick = onOpenPendingCorrections)
                MenuRow(icon = Icons.Outlined.Shield, title = "忌口管理", onClick = onOpenDietary)
            }
        }

        item {
            TextButton(onClick = onLogout, modifier = Modifier.fillMaxWidth()) {
                Text("退出登录", color = MaterialTheme.colorScheme.error)
            }
        }
    }
}

@Composable
private fun EditNicknameDialog(
    currentNickname: String,
    onDismiss: () -> Unit,
    onConfirm: (String) -> Unit,
) {
    var nickname by rememberSaveable { mutableStateOf(currentNickname) }
    AlertDialog(
        onDismissRequest = onDismiss,
        title = { Text("编辑昵称") },
        text = {
            OutlinedTextField(
                value = nickname,
                onValueChange = { nickname = it },
                singleLine = true,
                label = { Text("昵称") },
            )
        },
        confirmButton = {
            TextButton(
                onClick = { onConfirm(nickname.trim()) },
                enabled = nickname.isNotBlank(),
            ) {
                Text("保存")
            }
        },
        dismissButton = {
            TextButton(onClick = onDismiss) { Text("取消") }
        },
    )
}

@Composable
private fun MenuRow(icon: ImageVector, title: String, onClick: () -> Unit) {
    Row(
        verticalAlignment = Alignment.CenterVertically,
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick)
            .padding(horizontal = 16.dp, vertical = 14.dp)
    ) {
        Icon(
            icon,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary
        )
        Spacer(modifier = Modifier.width(12.dp))
        Text(
            text = title,
            style = MaterialTheme.typography.bodyLarge,
            modifier = Modifier.weight(1f)
        )
        Icon(
            Icons.AutoMirrored.Filled.KeyboardArrowRight,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.onSurfaceVariant
        )
    }
}
