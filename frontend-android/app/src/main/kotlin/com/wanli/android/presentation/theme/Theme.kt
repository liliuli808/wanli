package com.wanli.android.presentation.theme

import android.app.Activity
import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.SideEffect
import androidx.compose.ui.graphics.toArgb
import androidx.compose.ui.platform.LocalView
import androidx.core.view.WindowCompat

private val LightColorScheme = lightColorScheme(
    primary = Sage40,
    onPrimary = CardWhite,
    primaryContainer = Sage90,
    onPrimaryContainer = SageOnContainer,
    secondary = NeutralSecondary,
    onSecondary = CardWhite,
    secondaryContainer = NeutralSecondaryContainer,
    onSecondaryContainer = InkPrimary,
    tertiary = MustardDeep,
    onTertiary = CardWhite,
    tertiaryContainer = MustardSoft,
    onTertiaryContainer = MustardOnAccent,
    error = ErrorRed40,
    onError = CardWhite,
    errorContainer = ErrorRed90,
    onErrorContainer = InkPrimary,
    background = CreamBackground,
    onBackground = InkPrimary,
    surface = CreamBackground,
    onSurface = InkPrimary,
    surfaceVariant = CreamSurfaceVariant,
    onSurfaceVariant = InkSecondary,
    outline = OutlineGray,
    outlineVariant = OutlineVariantGray,
)

@Composable
fun WanLiTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit
) {
    // 本期不做暗黑模式（zakka 浅色风格先行），统一使用浅色方案。
    val colorScheme = LightColorScheme
    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as Activity).window
            window.statusBarColor = colorScheme.surface.toArgb()
            WindowCompat.getInsetsController(window, view).isAppearanceLightStatusBars = true
        }
    }
    MaterialTheme(
        colorScheme = colorScheme,
        typography = Typography,
        shapes = WanLiShapes,
        content = content
    )
}
