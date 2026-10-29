package com.wanli.android.presentation.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

// 暖调食欲系：标题无衬线粗体（对应 Noto Sans SC Bold），正文常规；不再使用衬线体。
private val TitleFamily = FontFamily.SansSerif
private val BodyFamily = FontFamily.SansSerif

val Typography = Typography(
    displaySmall = TextStyle(
        fontFamily = TitleFamily, fontWeight = FontWeight.Bold,
        fontSize = 32.sp, lineHeight = 40.sp, letterSpacing = (-0.3).sp,
    ),
    headlineMedium = TextStyle(
        fontFamily = TitleFamily, fontWeight = FontWeight.Bold,
        fontSize = 28.sp, lineHeight = 35.sp, letterSpacing = (-0.3).sp,
    ),
    headlineSmall = TextStyle(
        fontFamily = TitleFamily, fontWeight = FontWeight.Bold,
        fontSize = 22.sp, lineHeight = 28.sp,
    ),
    titleLarge = TextStyle(
        fontFamily = TitleFamily, fontWeight = FontWeight.Bold,
        fontSize = 20.sp, lineHeight = 26.sp,
    ),
    titleMedium = TextStyle(
        fontFamily = TitleFamily, fontWeight = FontWeight.Bold,
        fontSize = 18.sp, lineHeight = 24.sp,
    ),
    titleSmall = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.SemiBold,
        fontSize = 15.sp, lineHeight = 21.sp,
    ),
    bodyLarge = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Normal,
        fontSize = 15.sp, lineHeight = 23.sp,
    ),
    bodyMedium = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Normal,
        fontSize = 14.sp, lineHeight = 21.sp,
    ),
    bodySmall = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Normal,
        fontSize = 12.sp, lineHeight = 17.sp,
    ),
    labelLarge = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.SemiBold,
        fontSize = 15.sp, lineHeight = 20.sp,
    ),
    labelMedium = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Medium,
        fontSize = 13.sp, lineHeight = 18.sp,
    ),
    labelSmall = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Medium,
        fontSize = 12.sp, lineHeight = 16.sp,
    ),
)
