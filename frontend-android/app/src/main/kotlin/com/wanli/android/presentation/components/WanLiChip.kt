package com.wanli.android.presentation.components

import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.wanli.android.presentation.theme.CardWhite
import com.wanli.android.presentation.theme.InkPrimary
import com.wanli.android.presentation.theme.OnTomato
import com.wanli.android.presentation.theme.OutlineWarm
import com.wanli.android.presentation.theme.TomatoRed

/** 暖调食欲系筛选 chip：未选白底细描边，选中番茄红实心白字，全圆胶囊。 */
@Composable
fun WanLiFilterChip(
    selected: Boolean,
    onClick: () -> Unit,
    label: String,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    FilterChip(
        selected = selected,
        onClick = onClick,
        enabled = enabled,
        modifier = modifier,
        label = { Text(label, style = MaterialTheme.typography.labelMedium) },
        shape = CircleShape,
        colors = FilterChipDefaults.filterChipColors(
            containerColor = CardWhite,
            labelColor = InkPrimary,
            selectedContainerColor = TomatoRed,
            selectedLabelColor = OnTomato,
        ),
        border = FilterChipDefaults.filterChipBorder(
            enabled = enabled,
            selected = selected,
            borderColor = OutlineWarm,
            selectedBorderColor = TomatoRed,
        ),
    )
}
