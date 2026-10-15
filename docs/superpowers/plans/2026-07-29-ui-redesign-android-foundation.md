# 碗里 UI 重设计 · Android 地基（主题/组件/导航）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 Stitch 派生设计系统落到 Compose 主题，建立去 emoji 的共享组件库，接入 4 个新后端端点，完成 5 Tab 导航重构（消息替换忌口）。

**Architecture:** 单一 `WanLiTheme` 浅色主题（本期不做暗黑）；共享组件集中在 `presentation/components/`；数据层沿用 ApiService + Repository 薄封装；导航保持外层 NavHost + MainScreen 内层 Tab NavHost 两层结构。

**Tech Stack:** Kotlin 2.0, Compose BOM 2024.06.00, Material 3, material-icons-extended, Coil, Hilt, Retrofit + kotlinx.serialization, mockk + turbine（VM 测试）。

**前置依赖：** 后端计划 `2026-07-29-ui-redesign-backend.md` 的 Task 1-4 已上线（或本地可连）。Task 1-4 可以先行（不依赖后端），Task 5（数据层）起依赖后端契约。

**注意：** 项目当前不是 git 仓库，提交步骤为可选检查点。

**工作目录：** `/home/jichi/wanli/frontend-android`。构建 `./gradlew :app:assembleDebug`，单测 `./gradlew :app:testDebugUnitTest`。

**设计基准：** Stitch 项目 `projects/9874479371737599731`，派生设计系统 `assets/83999fb829444300b0e64a952ab9ad49`（色板/字体/间距以其 YAML token 为准）。规格文档 `docs/superpowers/specs/2026-07-29-ui-redesign-design.md` §4。

---

### Task 1: 主题 Token 落地（Color/Theme/Type + 资源色）

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/theme/Color.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/theme/Theme.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/theme/Type.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/components/StatusBanner.kt`
- Modify: `app/src/main/res/values/colors.xml`
- Shape.kt 不动（small=8 / medium=16 / large=24 已符合设计）

- [ ] **Step 1: 重写 Color.kt 为清新自然系色板**

整文件替换为：

```kotlin
package com.wanli.android.presentation.theme

import androidx.compose.ui.graphics.Color

// —— 鼠尾草绿（主色，Stitch derived DS）——
val Sage40 = Color(0xFF436645)          // primary
val SageOnContainer = Color(0xFF002108) // onPrimaryContainer
val Sage90 = Color(0xFFC5EDC3)          // primaryContainer
val SageSoft = Color(0xFFE8F0E8)        // 浅绿横幅/空状态底衬
val SageBrand = Color(0xFF5B7F5C)       // 品牌强调（徽章描边等）

// —— 米白（背景/表面）——
val CreamBackground = Color(0xFFFCF9F8)
val CreamSurfaceLow = Color(0xFFF6F3F2)
val CreamSurfaceVariant = Color(0xFFE4E2E1)
val CardWhite = Color(0xFFFFFFFF)

// —— 中性文字/描边 ——
val InkPrimary = Color(0xFF1B1C1C)
val InkSecondary = Color(0xFF424841)
val OutlineGray = Color(0xFF727970)
val OutlineVariantGray = Color(0xFFC2C8BE)

// —— 芥末黄（社交高光：评分/徽章/AI 推测）——
val MustardAccent = Color(0xFFD9A441)
val MustardOnAccent = Color(0xFF271900)
val MustardSoft = Color(0xFFFFDEAA)
val MustardDeep = Color(0xFF7C5700)

// —— 中性灰（M3 secondary 槽位）——
val NeutralSecondary = Color(0xFF5F5E59)
val NeutralSecondaryContainer = Color(0xFFE5E2DB)

// —— 语义色（StatusBanner 等使用）——
val ErrorRed40 = Color(0xFFBA1A1A)
val ErrorRed90 = Color(0xFFFFDAD6)
val SuccessGreen40 = Color(0xFF4A7A4F)
val SuccessGreen90 = Color(0xFFE8F0E8)
val WarningAmber40 = Color(0xFFC98A2D)
val WarningAmber90 = Color(0xFFFFF3E0)
val InfoSage40 = Sage40
val InfoSage90 = SageSoft
```

- [ ] **Step 2: 改 StatusBanner 引用**

`StatusBanner.kt` 把 `Terracotta20/90` 等旧色引用替换为新 token：`INFO → InfoSage40/InfoSage90`，`SUCCESS → SuccessGreen40/SuccessGreen90`，`WARNING → WarningAmber40/WarningAmber90`，`ERROR → ErrorRed40/ErrorRed90`。枚举结构不变。

- [ ] **Step 3: 重写 Theme.kt（本期锁浅色）**

整文件替换为：

```kotlin
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
```

- [ ] **Step 4: Type.kt 引入衬线标题层级**

整文件替换为：

```kotlin
package com.wanli.android.presentation.theme

import androidx.compose.material3.Typography
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.sp

// 标题衬线（对应 Stitch 的 Source Serif 4；中文回退系统衬线），正文系统无衬线（对应 Be Vietnam Pro）。
private val HeadlineFamily = FontFamily.Serif
private val BodyFamily = FontFamily.Default

val Typography = Typography(
    displaySmall = TextStyle(
        fontFamily = HeadlineFamily, fontWeight = FontWeight.Bold,
        fontSize = 28.sp, lineHeight = 36.sp,
    ),
    headlineMedium = TextStyle(
        fontFamily = HeadlineFamily, fontWeight = FontWeight.SemiBold,
        fontSize = 24.sp, lineHeight = 32.sp,
    ),
    headlineSmall = TextStyle(
        fontFamily = HeadlineFamily, fontWeight = FontWeight.SemiBold,
        fontSize = 20.sp, lineHeight = 28.sp,
    ),
    titleLarge = TextStyle(
        fontFamily = HeadlineFamily, fontWeight = FontWeight.SemiBold,
        fontSize = 18.sp, lineHeight = 24.sp,
    ),
    titleMedium = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.SemiBold,
        fontSize = 16.sp, lineHeight = 22.sp,
    ),
    titleSmall = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Medium,
        fontSize = 14.sp, lineHeight = 20.sp,
    ),
    bodyLarge = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Normal,
        fontSize = 16.sp, lineHeight = 24.sp,
    ),
    bodyMedium = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Normal,
        fontSize = 14.sp, lineHeight = 20.sp,
    ),
    bodySmall = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Normal,
        fontSize = 12.sp, lineHeight = 16.sp,
    ),
    labelLarge = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Medium,
        fontSize = 14.sp, lineHeight = 20.sp,
    ),
    labelSmall = TextStyle(
        fontFamily = BodyFamily, fontWeight = FontWeight.Medium,
        fontSize = 12.sp, lineHeight = 16.sp, letterSpacing = 0.5.sp,
    ),
)
```

- [ ] **Step 5: 更新 res/values/colors.xml**

```xml
<?xml version="1.0" encoding="utf-8"?>
<resources>
    <color name="sage_primary">#FF436645</color>
    <color name="cream_background">#FFFCF9F8</color>
</resources>
```

（`themes.xml` 的 `windowBackground` 引用 `@color/cream_background`，名字不变、值已更新，无需改 themes.xml；删掉不再被引用的旧 terracotta 色值。）

- [ ] **Step 6: 全项目编译，清理旧色引用**

Run: `./gradlew :app:assembleDebug`
Expected: 若其它文件仍 import `Terracotta40` 等旧 val 会编译报错——逐个替换为 `colorScheme.primary` 或上表新 token（优先走 colorScheme 语义槽）。最终 BUILD SUCCESSFUL。

- [ ] **Step 7: 提交检查点（可选）**

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/theme app/src/main/kotlin/com/wanli/android/presentation/components/StatusBanner.kt app/src/main/res/values/colors.xml
git commit -m "feat(theme): switch to fresh-natural sage/cream/mustard design tokens"
```

---

### Task 2: 引入 material-icons-extended

**Files:**
- Modify: `gradle/libs.versions.toml`
- Modify: `app/build.gradle.kts`

- [ ] **Step 1: 声明依赖**

`gradle/libs.versions.toml` 的 `[libraries]` 段加：

```toml
material-icons-extended = { group = "androidx.compose.material", name = "material-icons-extended", version = "1.6.8" }
```

`app/build.gradle.kts` 的 `dependencies` 加：

```kotlin
implementation(libs.material.icons.extended)
```

- [ ] **Step 2: 构建验证**

Run: `./gradlew :app:assembleDebug`
Expected: BUILD SUCCESSFUL（注意首次同步会下载较大 artifact；若 release 构建后续嫌包体积大，可再开 R8 资源压缩，本期不处理）

- [ ] **Step 3: 提交检查点（可选）** `git commit -m "build: add material-icons-extended for outlined icon system"`

---

### Task 3: 共享组件库（去 emoji 的积木）

**Files:**
- Create: `app/src/main/kotlin/com/wanli/android/presentation/components/WanLiCard.kt`
- Create: `app/src/main/kotlin/com/wanli/android/presentation/components/Social.kt`
- Create: `app/src/main/kotlin/com/wanli/android/presentation/components/SectionHeader.kt`
- Create: `app/src/main/kotlin/com/wanli/android/presentation/components/EmptyState.kt`
- Create: `app/src/main/kotlin/com/wanli/android/presentation/components/Skeleton.kt`
- Create: `app/src/main/kotlin/com/wanli/android/presentation/components/ImportBar.kt`

- [ ] **Step 1: WanLiCard.kt——白卡 + Soft Glow 投影**

```kotlin
package com.wanli.android.presentation.components

import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.theme.CardWhite

/** 设计系统标准卡片：纯白、16dp 圆角、柔和投影。 */
@Composable
fun WanLiCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val colors = CardDefaults.cardColors(containerColor = CardWhite)
    val elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    val shape = MaterialTheme.shapes.medium
    if (onClick != null) {
        Card(onClick = onClick, modifier = modifier, shape = shape, colors = colors, elevation = elevation, content = content)
    } else {
        Card(modifier = modifier, shape = shape, colors = colors, elevation = elevation, content = content)
    }
}

/** 卡片内容统一内边距修饰符。 */
fun Modifier.cardContentPadding(): Modifier = this.padding(16.dp)
```

- [ ] **Step 2: Social.kt——头像/作者行/评分星/社交徽章**

```kotlin
package com.wanli.android.presentation.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Star
import androidx.compose.material.icons.outlined.StarOutline
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import coil.compose.AsyncImage
import com.wanli.android.presentation.theme.MustardAccent
import com.wanli.android.presentation.theme.MustardOnAccent
import com.wanli.android.presentation.theme.SageSoft

/** 圆形用户头像：有图显示图，无图显示昵称首字（鼠尾草绿底）。 */
@Composable
fun UserAvatar(
    nickname: String,
    avatarUrl: String?,
    size: Dp = 36.dp,
    modifier: Modifier = Modifier,
) {
    if (!avatarUrl.isNullOrBlank()) {
        AsyncImage(
            model = avatarUrl,
            contentDescription = nickname,
            modifier = modifier.size(size).clip(CircleShape),
            contentScale = ContentScale.Crop,
        )
    } else {
        Box(
            modifier = modifier.size(size).clip(CircleShape).background(SageSoft),
            contentAlignment = Alignment.Center,
        ) {
            Text(
                text = nickname.firstOrNull()?.toString() ?: "碗",
                style = MaterialTheme.typography.titleSmall,
                color = MaterialTheme.colorScheme.primary,
            )
        }
    }
}

/** 作者行：头像 + 昵称 + 可选副信息（时间等）。 */
@Composable
fun AuthorRow(
    nickname: String,
    modifier: Modifier = Modifier,
    avatarUrl: String? = null,
    subtitle: String? = null,
    trailing: (@Composable RowScope.() -> Unit)? = null,
) {
    Row(modifier = modifier, verticalAlignment = Alignment.CenterVertically) {
        UserAvatar(nickname = nickname, avatarUrl = avatarUrl, size = 32.dp)
        Spacer(Modifier.width(8.dp))
        Column(Modifier.weight(1f)) {
            Text(
                text = nickname,
                style = MaterialTheme.typography.titleSmall,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )
            if (subtitle != null) {
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis,
                )
            }
        }
        trailing?.invoke(this)
    }
}

/** 评分星（芥末黄）。rating 为 null 时不渲染。 */
@Composable
fun RatingStars(rating: Int?, modifier: Modifier = Modifier, starSize: Dp = 16.dp) {
    if (rating == null) return
    Row(modifier = modifier) {
        repeat(5) { index ->
            Icon(
                imageVector = if (index < rating) Icons.Filled.Star else Icons.Outlined.StarOutline,
                contentDescription = null,
                tint = MustardAccent,
                modifier = Modifier.size(starSize),
            )
        }
    }
}

/** 芥末黄社交徽章，如「12 人做过」「AI 推测」。 */
@Composable
fun SocialBadge(text: String, modifier: Modifier = Modifier) {
    Surface(
        modifier = modifier,
        shape = MaterialTheme.shapes.small,
        color = MustardAccent,
    ) {
        Text(
            text = text,
            style = MaterialTheme.typography.labelSmall,
            fontWeight = FontWeight.Bold,
            color = MustardOnAccent,
            modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp),
        )
    }
}
```

- [ ] **Step 3: SectionHeader.kt——衬线分区标题（替代 emoji 标题）**

```kotlin
package com.wanli.android.presentation.components

import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.RowScope
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp

/** 衬线分区标题，如「食材」「步骤」「做过墙」。 */
@Composable
fun SectionHeader(
    title: String,
    modifier: Modifier = Modifier,
    trailing: (@Composable RowScope.() -> Unit)? = null,
) {
    Row(
        modifier = modifier.padding(vertical = 8.dp),
        verticalAlignment = Alignment.CenterVertically,
    ) {
        Text(text = title, style = MaterialTheme.typography.headlineSmall, modifier = Modifier.weight(1f))
        trailing?.invoke(this)
    }
}
```

- [ ] **Step 4: EmptyState.kt——碗筷插画空状态**

```kotlin
package com.wanli.android.presentation.components

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.vector.ImageVector
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.theme.SageSoft

/** 居中空状态：浅绿圆底图标 + 衬线标题 + 说明 + 主/次按钮（均可选）。 */
@Composable
fun EmptyState(
    icon: ImageVector,
    title: String,
    subtitle: String,
    modifier: Modifier = Modifier,
    primaryLabel: String? = null,
    onPrimary: (() -> Unit)? = null,
    secondaryLabel: String? = null,
    onSecondary: (() -> Unit)? = null,
) {
    Column(
        modifier = modifier.fillMaxWidth().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Box(
            modifier = Modifier.size(120.dp).clip(CircleShape).background(SageSoft),
            contentAlignment = Alignment.Center,
        ) {
            Icon(icon, contentDescription = null, tint = MaterialTheme.colorScheme.primary, modifier = Modifier.size(48.dp))
        }
        Spacer(Modifier.height(24.dp))
        Text(title, style = MaterialTheme.typography.headlineSmall, textAlign = TextAlign.Center)
        Spacer(Modifier.height(8.dp))
        Text(
            subtitle,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
        )
        if (primaryLabel != null && onPrimary != null) {
            Spacer(Modifier.height(24.dp))
            Button(onClick = onPrimary, shape = MaterialTheme.shapes.large, modifier = Modifier.fillMaxWidth()) {
                Text(primaryLabel)
            }
        }
        if (secondaryLabel != null && onSecondary != null) {
            Spacer(Modifier.height(8.dp))
            OutlinedButton(onClick = onSecondary, shape = MaterialTheme.shapes.large, modifier = Modifier.fillMaxWidth()) {
                Text(secondaryLabel)
            }
        }
    }
}
```

- [ ] **Step 5: Skeleton.kt——加载骨架屏**

```kotlin
package com.wanli.android.presentation.components

import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.unit.dp
import com.wanli.android.presentation.theme.CreamSurfaceVariant
import com.wanli.android.presentation.theme.CardWhite

/** 呼吸骨架卡片：列表加载态占位。 */
@Composable
fun SkeletonCard(modifier: Modifier = Modifier) {
    val transition = rememberInfiniteTransition(label = "skeleton")
    val alpha by transition.animateFloat(
        initialValue = 0.4f, targetValue = 1f,
        animationSpec = infiniteRepeatable(animation = tween(800), repeatMode = RepeatMode.Reverse),
        label = "alpha",
    )
    Column(modifier.fillMaxWidth().clip(MaterialTheme.shapes.medium).background(CardWhite).padding(16.dp)) {
        Box(Modifier.fillMaxWidth().height(160.dp).clip(MaterialTheme.shapes.small).background(CreamSurfaceVariant.copy(alpha = alpha)))
        Spacer(Modifier.height(12.dp))
        Box(Modifier.fillMaxWidth(0.6f).height(16.dp).clip(MaterialTheme.shapes.small).background(CreamSurfaceVariant.copy(alpha = alpha)))
        Spacer(Modifier.height(8.dp))
        Box(Modifier.fillMaxWidth(0.4f).height(14.dp).clip(MaterialTheme.shapes.small).background(CreamSurfaceVariant.copy(alpha = alpha)))
    }
}
```

- [ ] **Step 6: ImportBar.kt——搜索栏式视频导入入口**

```kotlin
package com.wanli.android.presentation.components

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowForward
import androidx.compose.material.icons.outlined.Link
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp

/** 首页顶部「粘贴 B 站链接」导入条。 */
@Composable
fun ImportBar(
    url: String,
    onUrlChange: (String) -> Unit,
    onSubmit: () -> Unit,
    modifier: Modifier = Modifier,
    enabled: Boolean = true,
) {
    WanLiCard(modifier = modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier.fillMaxWidth().padding(start = 16.dp, end = 8.dp, top = 8.dp, bottom = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Icon(Icons.Outlined.Link, contentDescription = null, tint = MaterialTheme.colorScheme.onSurfaceVariant)
            Spacer(Modifier.width(8.dp))
            TextField(
                value = url,
                onValueChange = onUrlChange,
                modifier = Modifier.weight(1f),
                placeholder = { Text("粘贴 B 站视频链接，AI 帮你解析菜谱") },
                singleLine = true,
                enabled = enabled,
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Go),
                keyboardActions = KeyboardActions(onGo = { onSubmit() }),
                colors = TextFieldDefaults.colors(
                    focusedContainerColor = androidx.compose.ui.graphics.Color.Transparent,
                    unfocusedContainerColor = androidx.compose.ui.graphics.Color.Transparent,
                    focusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                    unfocusedIndicatorColor = androidx.compose.ui.graphics.Color.Transparent,
                ),
            )
            FilledIconButton(onClick = onSubmit, enabled = enabled && url.isNotBlank()) {
                Icon(Icons.AutoMirrored.Filled.ArrowForward, contentDescription = "导入")
            }
        }
    }
}
```

- [ ] **Step 7: 构建验证 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug`
Expected: BUILD SUCCESSFUL

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/components/
git commit -m "feat(components): add redesign shared component library (cards, social, empty, skeleton, import bar)"
```

---

### Task 4: 数据层接入新端点

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/data/remote/ApiService.kt`
- Create: `app/src/main/kotlin/com/wanli/android/data/repository/SocialRepository.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/data/repository/AuthRepository.kt`

- [ ] **Step 1: ApiService 加 DTO 与端点**

在 `ApiService.kt` 的 interface 内追加（放在 `me()` 附近）：

```kotlin
@PUT("me/profile")
suspend fun updateProfile(@Body request: UpdateProfileRequest): MeResponse

@GET("feed/cooking-records")
suspend fun getCookingFeed(@Query("limit") limit: Int): CookingFeedResponse

@GET("me/messages")
suspend fun getMessages(): MessagesResponse

@GET("me/cooking-records")
suspend fun getMyCookingRecords(@Query("limit") limit: Int): MyCookingRecordsResponse
```

文件内 DTO 区追加：

```kotlin
@Serializable
data class UpdateProfileRequest(
    val nickname: String,
    @SerialName("avatar_url") val avatarUrl: String? = null,
)

@Serializable
data class CookingFeedItemDto(
    val id: String,
    val rating: Int? = null,
    @SerialName("photo_url") val photoUrl: String? = null,
    val feedback: String? = null,
    @SerialName("author_nickname") val authorNickname: String = "",
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("recipe_version_id") val recipeVersionId: String,
    @SerialName("recipe_id") val recipeId: String,
    @SerialName("recipe_title") val recipeTitle: String,
    @SerialName("cover_url") val coverUrl: String? = null,
)

@Serializable
data class CookingFeedData(val items: List<CookingFeedItemDto> = emptyList())

@Serializable
data class CookingFeedResponse(
    val success: Boolean,
    val data: CookingFeedData? = null,
    val error: ErrorBody? = null,
)

@Serializable
data class MessageDto(
    val type: String,
    val id: String,
    @SerialName("actor_nickname") val actorNickname: String = "",
    val text: String? = null,
    val rating: Int? = null,
    @SerialName("recipe_version_id") val recipeVersionId: String? = null,
    @SerialName("recipe_title") val recipeTitle: String = "",
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class MessagesData(val items: List<MessageDto> = emptyList())

@Serializable
data class MessagesResponse(
    val success: Boolean,
    val data: MessagesData? = null,
    val error: ErrorBody? = null,
)

@Serializable
data class MyCookingRecordDto(
    val id: String,
    val rating: Int? = null,
    @SerialName("photo_url") val photoUrl: String? = null,
    val feedback: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("recipe_version_id") val recipeVersionId: String,
    @SerialName("recipe_title") val recipeTitle: String,
    @SerialName("cover_url") val coverUrl: String? = null,
)

@Serializable
data class MyCookingRecordsData(val items: List<MyCookingRecordDto> = emptyList())

@Serializable
data class MyCookingRecordsResponse(
    val success: Boolean,
    val data: MyCookingRecordsData? = null,
    val error: ErrorBody? = null,
)
```

同时修改现有 `MeResponse` 的 data 结构：找到 `MeResponse`/`MeData`（`me(): MeResponse` 的返回类型），确保其 data 含 `nickname: String = ""` 与 `@SerialName("avatar_url") val avatarUrl: String? = null` 字段（现有 UserDto 已有这两字段，若 MeData 是独立结构则补齐）。

`RecipeSummaryDto` 追加字段：

```kotlin
@SerialName("cooking_record_count") val cookingRecordCount: Int = 0,
```

- [ ] **Step 2: SocialRepository**

```kotlin
package com.wanli.android.data.repository

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.CookingFeedItemDto
import com.wanli.android.data.remote.MessageDto
import com.wanli.android.data.remote.MyCookingRecordDto
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class SocialRepository @Inject constructor(
    private val apiService: ApiService,
) {
    suspend fun getCookingFeed(limit: Int = 20): ApiResult<List<CookingFeedItemDto>> = wrap("加载动态失败") {
        apiService.getCookingFeed(limit).data?.items ?: emptyList()
    }

    suspend fun getMessages(): ApiResult<List<MessageDto>> = wrap("加载消息失败") {
        apiService.getMessages().data?.items ?: emptyList()
    }

    suspend fun getMyCookingRecords(limit: Int = 60): ApiResult<List<MyCookingRecordDto>> = wrap("加载做过失败") {
        apiService.getMyCookingRecords(limit).data?.items ?: emptyList()
    }
}
```

（`wrap` 辅助函数的签名以 `RecipeRepository` 里现有的为准——照抄它的私有 `wrap` 写法；若它是 repository 内私有函数，就在本文件复制一份同样的实现。）

- [ ] **Step 3: AuthRepository 加 updateProfile 与 me**

照 `AuthRepository` 现有风格追加：

```kotlin
suspend fun getMe(): ApiResult<UserDto> = wrap("加载用户信息失败") {
    apiService.me().data ?: throw IllegalStateException("empty me response")
}

suspend fun updateProfile(nickname: String, avatarUrl: String?): ApiResult<UserDto> = wrap("更新资料失败") {
    apiService.updateProfile(UpdateProfileRequest(nickname, avatarUrl)).data ?: throw IllegalStateException("empty update response")
}
```

（返回 data 的实际类型以 `MeResponse` 的 data 类型为准——若是 `MeData` 而非 `UserDto`，签名相应改为 `ApiResult<MeData>`。）

- [ ] **Step 4: 构建验证 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug`
Expected: BUILD SUCCESSFUL

```bash
git add app/src/main/kotlin/com/wanli/android/data/
git commit -m "feat(data): wire profile/feed/messages/my-cooking endpoints"
```

---

### Task 5: 导航重构——5 Tab（消息上位，忌口降级）

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/main/MainScreen.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/navigation/WanLiNavHost.kt`
- Create: `app/src/main/kotlin/com/wanli/android/presentation/messages/MessagesScreen.kt`（本任务先放占位页，完整实现在 screens 计划 Task 7）

- [ ] **Step 1: MainTab 换成新 5 Tab**

`MainScreen.kt` 的 `sealed class MainTab` 替换为：

```kotlin
sealed class MainTab(val route: String, val title: String, val icon: ImageVector) {
    data object Home : MainTab("home", "首页", Icons.Outlined.Home)
    data object Discover : MainTab("discover", "发现", Icons.Outlined.Explore)
    data object Kitchen : MainTab("kitchen", "厨房", Icons.Outlined.Kitchen)
    data object Messages : MainTab("messages", "消息", Icons.Outlined.Notifications)
    data object Profile : MainTab("profile", "我的", Icons.Outlined.Person)
}
```

`tabs` 列表更新为 `listOf(MainTab.Home, MainTab.Discover, MainTab.Kitchen, MainTab.Messages, MainTab.Profile)`。

- [ ] **Step 2: 内层 NavHost 换路由**

`MainScreen.kt` 内层 `NavHost` 中：删除 `composable(MainTab.Dietary.route)` 分支，新增：

```kotlin
composable(MainTab.Messages.route) {
    if (isLoggedIn) {
        MessagesScreen(onOpenRecipe = onOpenRecipe)
    } else {
        LoginRequired(onLogin = onLogin)
    }
}
```

（`onOpenRecipe: (String) -> Unit` 参数加入 `MainScreen` 签名，由外层传入 `navController.navigate(Routes.recipeDetail(versionId))`；`onLogin` 沿用现有登引导航逻辑——若现有 `LoginRequired` 无参数则保持无参。）

- [ ] **Step 3: 忌口降级为「我的」内的二级路由**

`WanLiNavHost.kt` 的 `Routes` object 加：

```kotlin
const val DIETARY = "me/dietary-restrictions"
```

`WanLiNavHost` 中加：

```kotlin
composable(Routes.DIETARY) {
    DietaryScreen(onBack = { navController.popBackStack() })
}
```

`DietaryScreen` 加 `onBack: () -> Unit` 参数并用 `WanLiTopBar(title = "忌口管理", onBack = onBack)` 包裹（替换其现有自绘 header）。import 同步更新。

- [ ] **Step 4: 占位 MessagesScreen**

```kotlin
package com.wanli.android.presentation.messages

import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Scaffold
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import com.wanli.android.presentation.components.WanLiTopBar

/** 消息中心（占位，完整列表在后续任务实现）。 */
@Composable
fun MessagesScreen(onOpenRecipe: (String) -> Unit, modifier: Modifier = Modifier) {
    Scaffold(topBar = { WanLiTopBar(title = "消息") }) { padding ->
        Column(modifier.fillMaxSize().padding(padding)) { }
    }
}
```

- [ ] **Step 5: MainScreen 拼装修正**

确认 `MainScreen` 调用处（`WanLiNavHost` 的 `composable(Routes.MAIN)`）传入新的 `onOpenRecipe` 参数；Profile tab 需要新增菜单项「忌口管理」的入口回调 `onOpenDietary: () -> Unit`（同样在 MAIN 处传入 `navController.navigate(Routes.DIETARY)`）。`ProfileScreen` 签名加 `onOpenDietary: () -> Unit`，菜单区加一行 `MenuRow(icon = Icons.Outlined.Shield, title = "忌口管理", onClick = onOpenDietary)`（MenuRow 若当前是 emoji 图标参数，本任务先把该行用 Icon 版实现，完整 Profile 重写在 screens 计划 Task 6）。

- [ ] **Step 6: 构建 + 跑既有测试**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest`
Expected: BUILD SUCCESSFUL，既有测试 PASS（若有测试引用了 Dietary tab 路由则同步修正）

- [ ] **Step 7: 提交检查点（可选）**

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/
git commit -m "feat(nav): 5-tab IA with Messages tab; dietary demoted to profile subpage"
```

---

## 执行记录（2026-07-29）

5 个任务全部完成（实现 → 规格审查 → 质量审查循环通过）。过程中的重要修正：

- T1 质量审查修复：WARNING 横幅对比度 2.7:1 → WarningAmber40 改为 #7C5700（5.95:1）；SUCCESS 改为 Sage40/#DBEBD2（5.2:1，与 INFO 区分）；Type.kt 补 labelMedium。
- T3 质量审查修复：RatingStars 加语义 contentDescription + rating 钳制 0-5；ImportBar 补 disabled 透明色与 IME 空提交守卫。
- T4：删除 AuthRepository 死代码 safeCall。
- 后续项（非阻塞）：`wrap` 辅助函数已三处复制，第四次出现前提取为共享 helper（同时修 CancellationException 吞掉与 success=false 时丢弃服务端 message 的问题）；`login()` 无异常处理（既存问题）；ProfileScreen 菜单行 emoji 与 Icon 混排（界面 T6 统一）。

## Self-Review 记录

- 规格覆盖：§4 视觉 token（Task 1-3）、§3 IA 重构（Task 5）、§6 后端补充的客户端接入（Task 4）均有对应任务。暗黑模式本期非目标，Theme 锁浅色已注释说明。
- 占位说明：MessagesScreen 占位实现是刻意的分阶段产物（screens 计划 Task 7 补全），不是计划空洞。
- 图标可用性：本计划所用 `Icons.Outlined.*`（Home/Explore/Kitchen/Notifications/Person/Link/StarOutline/Shield）与 `Icons.Filled.Star`、`Icons.AutoMirrored.Filled.ArrowForward` 均在 material-icons-extended（Task 2 引入）中提供；若某个图标编译报 unresolved，用同语义近似图标替换即可。
