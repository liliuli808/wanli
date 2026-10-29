package com.wanli.android.presentation.components

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material3.HorizontalDivider
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.draw.shadow
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.theme.CardWhite
import com.wanli.android.presentation.theme.InkSecondary
import com.wanli.android.presentation.theme.OnTomato
import com.wanli.android.presentation.theme.OutlineWarm
import com.wanli.android.presentation.theme.TomatoRed
import com.wanli.android.presentation.theme.WarmShadow

/** 底部导航项：选中时使用填充图标。 */
data class BottomTab(
    val route: String,
    val title: String,
    val icon: ImageVector,
    val selectedIcon: ImageVector,
)

/**
 * 5 位置底部导航：首页 / 发现 / 中央导入 FAB / 消息 / 我的。
 * [tabs] 传 4 项，前两项在左、后两项在右，中央留出 FAB 位置。
 */
@Composable
fun AppBottomBar(
    tabs: List<BottomTab>,
    selectedRoute: String?,
    onSelect: (BottomTab) -> Unit,
    onCenterClick: () -> Unit,
    modifier: Modifier = Modifier,
    centerIcon: ImageVector = Icons.Filled.Add,
) {
    Box(modifier = modifier.fillMaxWidth()) {
        Column(Modifier.fillMaxWidth().background(CardWhite)) {
            HorizontalDivider(thickness = 1.dp, color = OutlineWarm)
            Row(
                modifier = Modifier.fillMaxWidth().height(64.dp).padding(horizontal = 8.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                tabs.take(2).forEach { tab ->
                    BottomTabItem(tab, tab.route == selectedRoute, { onSelect(tab) }, Modifier.weight(1f))
                }
                Box(Modifier.weight(1f))
                tabs.drop(2).forEach { tab ->
                    BottomTabItem(tab, tab.route == selectedRoute, { onSelect(tab) }, Modifier.weight(1f))
                }
            }
        }
        Box(
            modifier = Modifier
                .align(Alignment.TopCenter)
                .offset(y = (-20).dp)
                .size(56.dp)
                .shadow(10.dp, CircleShape, ambientColor = WarmShadow, spotColor = WarmShadow)
                .clip(CircleShape)
                .background(TomatoRed)
                .clickable(role = Role.Button, onClick = onCenterClick),
            contentAlignment = Alignment.Center,
        ) {
            Icon(centerIcon, contentDescription = "导入菜谱", tint = OnTomato, modifier = Modifier.size(28.dp))
        }
    }
}

@Composable
private fun BottomTabItem(
    tab: BottomTab,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .clip(MaterialTheme.shapes.medium)
            .clickable(role = Role.Tab, onClick = onClick)
            .padding(vertical = 6.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Icon(
            imageVector = if (selected) tab.selectedIcon else tab.icon,
            contentDescription = tab.title,
            tint = if (selected) TomatoRed else InkSecondary,
            modifier = Modifier.size(24.dp),
        )
        Text(
            text = tab.title,
            style = MaterialTheme.typography.labelSmall,
            color = if (selected) TomatoRed else InkSecondary,
            modifier = Modifier.padding(top = 2.dp),
        )
    }
}
