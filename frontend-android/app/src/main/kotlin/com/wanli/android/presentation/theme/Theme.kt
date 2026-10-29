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
    primary = TomatoRed,
    onPrimary = OnTomato,
    primaryContainer = TomatoContainer,
    onPrimaryContainer = OnTomatoContainer,
    secondary = CaramelAccent,
    onSecondary = CaramelOnAccent,
    secondaryContainer = CaramelSoft,
    onSecondaryContainer = CaramelOnAccent,
    tertiary = CaramelDeep,
    onTertiary = CardWhite,
    tertiaryContainer = CaramelSoft,
    onTertiaryContainer = CaramelOnAccent,
    error = ErrorRed,
    onError = CardWhite,
    errorContainer = ErrorRedContainer,
    onErrorContainer = InkPrimary,
    background = CreamBackground,
    onBackground = InkPrimary,
    surface = CardWhite,
    onSurface = InkPrimary,
    surfaceVariant = SurfaceWarm,
    onSurfaceVariant = InkSecondary,
    surfaceContainer = CreamBackground,
    surfaceContainerLow = CreamBackground,
    surfaceContainerHigh = CardWhite,
    surfaceContainerHighest = SurfaceWarm,
    outline = OutlineWarm,
    outlineVariant = OutlineVariantWarm,
)

@Composable
fun WanLiTheme(
    darkTheme: Boolean = isSystemInDarkTheme(),
    content: @Composable () -> Unit
) {
    // 本期不做暗黑模式（暖调食欲系浅色才成立），统一使用浅色方案。
    val colorScheme = LightColorScheme
    val view = LocalView.current
    if (!view.isInEditMode) {
        SideEffect {
            val window = (view.context as Activity).window
            window.statusBarColor = colorScheme.background.toArgb()
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
