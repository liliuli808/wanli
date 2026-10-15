# 碗里 UI 重设计 · Android 逐屏实现 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按 Stitch 设计稿逐屏重写 11 个界面：首页 Feed、发现瀑布流、详情社交化、版本谱系树、厨房、我的（身份+照片墙）、消息中心、登录、适配结果、校对编辑、全局收尾。

**Architecture:** 每屏 = Screen composable + Hilt ViewModel（`XxxUiState` 密封类 + StateFlow），UI 全部使用地基计划产出的共享组件（WanLiCard/SectionHeader/UserAvatar/AuthorRow/RatingStars/SocialBadge/EmptyState/SkeletonCard/ImportBar）。视觉基准为 Stitch 项目 `projects/9874479371737599731` 各屏截图。

**Tech Stack:** Compose BOM 2024.06.00 + material-icons-extended, Coil, Hilt, mockk + turbine（VM 测试）。

**前置依赖：** `2026-07-29-ui-redesign-android-foundation.md` 全部完成（主题/组件/数据层/导航已就位）；后端 4 个端点可用。

**注意：** 项目当前不是 git 仓库，提交步骤为可选检查点。工作目录 `/home/jichi/wanli/frontend-android`。

**通用规则（每个任务都适用，不再重复）：**
- 严禁新增 emoji；发现旧 emoji 一律换成 `Icons.Outlined.*` 图标或删除。
- 列表加载态用 `SkeletonCard`，错误态用 `ErrorMessage` + 重试，空态用 `EmptyState`。
- 现有 ViewModel 的成员名（状态字段、方法名）以代码为准；本计划给出的调用点若与现有命名不一致，按现有命名适配，不要改 VM 的对外契约（任务明确要求改的除外）。
- 每完成一屏：`./gradlew :app:assembleDebug` 必须 BUILD SUCCESSFUL，相关单测全绿。

---

### Task 1: 首页改造为社区 Feed

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/home/HomeViewModel.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/home/HomeScreen.kt`
- Test: `app/src/test/kotlin/com/wanli/android/presentation/home/HomeViewModelTest.kt`

**Stitch 基准：** 首页 `a8eb432a6f0047629ff10180cb142502` + 空状态 `86af40b39e804182b163fe48c6f8fe3f`。

- [ ] **Step 1: 重写 HomeViewModelTest（先红）**

整文件替换（沿用文件原有 `MainDispatcherRule`/dispatcher 设置方式；若旧测试用 `Dispatchers.setMain` 则照抄其 setup）：

```kotlin
package com.wanli.android.presentation.home

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.CookingFeedItemDto
import com.wanli.android.data.repository.SocialRepository
import io.mockk.coEvery
import io.mockk.mockk
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class HomeViewModelTest {

    private val socialRepository = mockk<SocialRepository>()

    private fun feedItem(id: String) = CookingFeedItemDto(
        id = id, rating = 5, photoUrl = null, feedback = "好吃",
        authorNickname = "小厨", createdAt = null,
        recipeVersionId = "v1", recipeId = "r1", recipeTitle = "红烧肉", coverUrl = null,
    )

    @Test
    fun `init loads feed successfully`() = runTest {
        coEvery { socialRepository.getCookingFeed(any()) } returns ApiResult.Success(listOf(feedItem("1")))

        val viewModel = HomeViewModel(socialRepository)

        val state = viewModel.feedState.value
        assertTrue(state is HomeFeedUiState.Success)
        assertEquals(1, (state as HomeFeedUiState.Success).items.size)
    }

    @Test
    fun `init shows error when feed fails`() = runTest {
        coEvery { socialRepository.getCookingFeed(any()) } returns ApiResult.Error(500, "网络错误")

        val viewModel = HomeViewModel(socialRepository)

        val state = viewModel.feedState.value
        assertTrue(state is HomeFeedUiState.Error)
        assertEquals("网络错误", (state as HomeFeedUiState.Error).message)
    }
}
```

（`ApiResult.Error` 的构造参数以 `data/remote/ApiResult.kt` 实际声明为准。）

- [ ] **Step 2: 运行确认失败**

Run: `./gradlew :app:testDebugUnitTest --tests "*HomeViewModelTest*"`
Expected: FAIL（`HomeFeedUiState` 不存在 / 构造函数不匹配）

- [ ] **Step 3: 重写 HomeViewModel**

整文件替换（健康检查药丸从首页移除）：

```kotlin
package com.wanli.android.presentation.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.CookingFeedItemDto
import com.wanli.android.data.repository.SocialRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

sealed interface HomeFeedUiState {
    data object Loading : HomeFeedUiState
    data class Error(val message: String) : HomeFeedUiState
    data class Success(val items: List<CookingFeedItemDto>) : HomeFeedUiState
}

@HiltViewModel
class HomeViewModel @Inject constructor(
    private val socialRepository: SocialRepository,
) : ViewModel() {

    private val _feedState = MutableStateFlow<HomeFeedUiState>(HomeFeedUiState.Loading)
    val feedState: StateFlow<HomeFeedUiState> = _feedState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _feedState.value = HomeFeedUiState.Loading
            when (val result = socialRepository.getCookingFeed()) {
                is ApiResult.Success -> _feedState.value = HomeFeedUiState.Success(result.data)
                is ApiResult.Error -> _feedState.value = HomeFeedUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }
}
```

- [ ] **Step 4: 运行确认通过**

Run: `./gradlew :app:testDebugUnitTest --tests "*HomeViewModelTest*"`
Expected: PASS

- [ ] **Step 5: 重写 HomeScreen**

整文件替换。导入交互复用现有 `ImportViewModel`（其 url 状态与 submit 方法名以现有代码为准，下文中 `importViewModel.url` / `importViewModel::onUrlChange` / `importViewModel::submit` 按实际适配；导入结果横幅沿用现有 `ImportUiState` 分支渲染，可复用 `StatusBanner`）：

```kotlin
package com.wanli.android.presentation.home

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.SoupKitchen
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import coil.compose.AsyncImage
import com.wanli.android.data.remote.CookingFeedItemDto
import com.wanli.android.presentation.components.*

@Composable
fun HomeScreen(
    onOpenRecipe: (String) -> Unit,
    onOpenMessages: () -> Unit,
    onOpenDiscover: () -> Unit,
    modifier: Modifier = Modifier,
    homeViewModel: HomeViewModel = hiltViewModel(),
    importViewModel: ImportViewModel = hiltViewModel(),
) {
    val feedState by homeViewModel.feedState.collectAsState()
    val url by importViewModel.url.collectAsState() // 以现有成员名为准

    LazyColumn(
        modifier = modifier.fillMaxSize(),
        contentPadding = PaddingValues(horizontal = 20.dp, vertical = 12.dp),
        verticalArrangement = Arrangement.spacedBy(16.dp),
    ) {
        item {
            Row(Modifier.fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                Text("碗里", style = MaterialTheme.typography.headlineMedium, modifier = Modifier.weight(1f))
                IconButton(onClick = onOpenMessages) {
                    Icon(Icons.Outlined.Notifications, contentDescription = "消息", tint = MaterialTheme.colorScheme.primary)
                }
            }
        }
        item {
            ImportBar(url = url, onUrlChange = importViewModel::onUrlChange, onSubmit = importViewModel::submit)
        }
        // 导入状态横幅：按现有 ImportUiState 分支用 StatusBanner 渲染（沿用旧 HomeScreen 的文案映射）
        when (val state = feedState) {
            is HomeFeedUiState.Loading -> items(3) { SkeletonCard() }
            is HomeFeedUiState.Error -> item {
                ErrorMessage(message = state.message)
                TextButton(onClick = homeViewModel::refresh) { Text("重试") }
            }
            is HomeFeedUiState.Success -> {
                if (state.items.isEmpty()) {
                    item {
                        EmptyState(
                            icon = Icons.Outlined.SoupKitchen,
                            title = "还没有菜谱动态",
                            subtitle = "导入第一道菜谱，或去看看社区里大家都在做什么",
                            primaryLabel = "粘贴链接导入",
                            onPrimary = importViewModel::submit,
                            secondaryLabel = "去发现页逛逛",
                            onSecondary = onOpenDiscover,
                        )
                    }
                } else {
                    items(state.items, key = { it.id }) { item ->
                        FeedCard(item = item, onClick = { onOpenRecipe(item.recipeVersionId) })
                    }
                }
            }
        }
    }
}

@Composable
private fun FeedCard(item: CookingFeedItemDto, onClick: () -> Unit) {
    WanLiCard(onClick = onClick) {
        val imageUrl = item.photoUrl ?: item.coverUrl
        if (imageUrl != null) {
            AsyncImage(
                model = imageUrl,
                contentDescription = item.recipeTitle,
                modifier = Modifier.fillMaxWidth().height(180.dp).clip(MaterialTheme.shapes.medium),
                contentScale = ContentScale.Crop,
            )
        }
        Column(Modifier.padding(top = 12.dp)) {
            AuthorRow(
                nickname = item.authorNickname,
                subtitle = "做了《${item.recipeTitle}》",
                trailing = { RatingStars(rating = item.rating) },
            )
            if (!item.feedback.isNullOrBlank()) {
                Spacer(Modifier.height(8.dp))
                Text(item.feedback, style = MaterialTheme.typography.bodyMedium)
            }
        }
    }
}
```

`MainScreen` 中 `composable(MainTab.Home.route)` 分支改为传入新参数（`onOpenMessages` = 切换到 Messages tab 的 inner nav 导航；`onOpenDiscover` 同理；`onOpenRecipe` 复用现有外传参数）。

- [ ] **Step 6: 构建 + 全量单测 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest`
Expected: 全绿。`HealthRepository` 若无其它引用则保留文件不删（后续排查用）。

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/home app/src/main/kotlin/com/wanli/android/presentation/main app/src/test/kotlin/com/wanli/android/presentation/home
git commit -m "feat(home): redesign as community cooked-it feed with import bar"
```

---

### Task 2: 发现页双列瀑布流

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/discover/DiscoverScreen.kt`
- Test: `app/src/test/kotlin/com/wanli/android/presentation/discover/DiscoverScreenTest.kt`（`discoverItemKey` 若仍被使用则测试不动）

**Stitch 基准：** `89b3c36b74454169be80b51d9856e076`。

- [ ] **Step 1: 列表换瀑布流 + 卡片重做**

`DiscoverScreen.kt`：保留 TabRow（最新/最热，选中色自动取自新主题 `colorScheme.primary`）与现有分页逻辑/ViewModel。把原来的 `LazyColumn` + 84dp 缩略图卡片替换为：

```kotlin
LazyVerticalStaggeredGrid(
    columns = StaggeredGridCells.Fixed(2),
    modifier = Modifier.fillMaxSize(),
    contentPadding = PaddingValues(horizontal = 16.dp, vertical = 8.dp),
    horizontalArrangement = Arrangement.spacedBy(12.dp),
    verticalItemSpacing = 12.dp,
) {
    items(items = recipes, key = { discoverItemKey(it) }) { recipe ->
        DiscoverCard(recipe = recipe, onClick = { onOpenRecipe(recipe.recipeVersionId) })
    }
    // 底部分页加载 item 沿用现有逻辑
}
```

卡片 composable（`DiscoverCard`，私有）：

```kotlin
@Composable
private fun DiscoverCard(recipe: RecipeSummaryDto, onClick: () -> Unit) {
    WanLiCard(onClick = onClick) {
        if (recipe.coverUrl != null) {
            AsyncImage(
                model = recipe.coverUrl,
                contentDescription = recipe.title,
                modifier = Modifier.fillMaxWidth().aspectRatio(1f).clip(MaterialTheme.shapes.medium),
                contentScale = ContentScale.Crop,
            )
        }
        Column(Modifier.padding(top = 8.dp)) {
            Text(recipe.title, style = MaterialTheme.typography.titleSmall, maxLines = 2, overflow = TextOverflow.Ellipsis)
            Spacer(Modifier.height(6.dp))
            AuthorRow(
                nickname = recipe.authorNickname.ifBlank { recipe.upNickname },
                trailing = {
                    when {
                        recipe.cookingRecordCount > 0 -> SocialBadge("${recipe.cookingRecordCount} 人做过")
                        recipe.publicVersionCount > 1 -> SocialBadge("${recipe.publicVersionCount} 个版本")
                    }
                },
            )
        }
    }
}
```

（字段名以 `RecipeSummaryDto` 实际为准：`cover_url`→`coverUrl`、`recipe_version_id`→`recipeVersionId` 等，序列化命名策略若全局 snake_case 则直接用现有访问方式。）空态/错误态的 `🍽️`/`😵` emoji 换成 `EmptyState(icon = Icons.Outlined.Restaurant, ...)`。

- [ ] **Step 2: 构建 + 测试 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest --tests "*Discover*"`
Expected: 全绿

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/discover
git commit -m "feat(discover): two-column waterfall with big covers and social badges"
```

---

### Task 3: 菜谱详情社交化重构

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/recipe/RecipeDetailScreen.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/recipe/CommunitySection.kt`

**Stitch 基准：** `cc303a33bc3643af806c6895b784f5ce`。

- [ ] **Step 1: 详情页结构重排**

按以下顺序重组 `RecipeDetailScreen` 内容区（数据仍走现有 `RecipeDetailViewModel` 的 state）：

1. 封面 `AsyncImage`（全宽、16:9、顶部圆角）
2. 标题（`typography.headlineMedium` 衬线）+ meta 行（份量/时长，用 `labelSmall` chip）
3. **版本谱系入口芯片**（替代原「版本树」按钮）：

```kotlin
Surface(
    onClick = { onOpenVersionTree(version.recipeId) },
    shape = MaterialTheme.shapes.small,
    color = MaterialTheme.colorScheme.secondaryContainer,
) {
    Row(Modifier.padding(horizontal = 12.dp, vertical = 8.dp), verticalAlignment = Alignment.CenterVertically) {
        Icon(Icons.Outlined.AccountTree, contentDescription = null, tint = MaterialTheme.colorScheme.primary)
        Spacer(Modifier.width(6.dp))
        Text(versionTypeLabel(version.versionType), style = MaterialTheme.typography.labelLarge)
        Icon(Icons.AutoMirrored.Filled.KeyboardArrowRight, contentDescription = null)
    }
}
```

4. 食材/步骤：`SectionHeader("食材")` / `SectionHeader("步骤")`（删掉 `🧂`/`👨‍🍳` emoji 与 `💡`），步骤时间戳用 `SocialBadge` 样式改为 `secondaryContainer` 小 chip
5. 操作行：`适配我的厨房 / 提交纠错 / 派生新版本` 改为一排 `OutlinedButton`（等宽、小字号），不再是满宽按钮堆叠
6. 底部 `CommunitySection`

- [ ] **Step 2: CommunitySection 加做过墙 + 评论带头像**

在评论列表**之前**插入「做过墙」（数据来自现有 cooking records state，过滤 `photoUrl != null`）：

```kotlin
val photoRecords = records.filter { !it.photoUrl.isNullOrBlank() }
if (photoRecords.isNotEmpty()) {
    SectionHeader("做过墙")
    LazyRow(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
        items(photoRecords, key = { it.id }) { record ->
            AsyncImage(
                model = record.photoUrl,
                contentDescription = null,
                modifier = Modifier.size(96.dp).clip(MaterialTheme.shapes.small),
                contentScale = ContentScale.Crop,
            )
        }
    }
}
```

评论行重做：`AuthorRow(nickname = comment.authorNickname, subtitle = comment.createdAt?.let(::formatTimestamp))` + 正文；做过记录行：`AuthorRow` + `RatingStars(record.rating)` + feedback；星级渲染删除 `★`/`☆` 字符拼接。错误/空态 emoji（`😵`）换 `EmptyState`。

- [ ] **Step 3: 构建 + 测试 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest --tests "*RecipeDetail*" --tests "*Community*"`
Expected: 全绿

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/recipe/RecipeDetailScreen.kt app/src/main/kotlin/com/wanli/android/presentation/recipe/CommunitySection.kt
git commit -m "feat(detail): social-first recipe detail with cooked-wall and lineage chip"
```

---

### Task 4: 版本谱系树状图

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/recipe/VersionTreeScreen.kt`

**Stitch 基准：** `e77d2c31535d41ce90d0f7cfacbee32f`。

- [ ] **Step 1: 朴素列表 → 缩进谱系树**

`VersionTreeScreen` 数据仍走现有 `VersionTreeViewModel`（nodes: `List<VersionTreeNodeDto>`）。内容区替换为：

```kotlin
val roots = nodes.filter { it.parentId == null }.sortedBy { it.createdAt }
val childrenOf: (String?) -> List<VersionTreeNodeDto> = { pid ->
    nodes.filter { it.parentId == pid }.sortedBy { it.createdAt }
}

LazyColumn(contentPadding = PaddingValues(20.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
    roots.forEach { root ->
        item { VersionNodeCard(node = root, depth = 0, isRoot = true, onClick = { onOpenVersion(root.id) }) }
        renderChildren(root.id, depth = 1)
    }
}
```

`renderChildren` 为 `LazyListScope` 扩展：递归 `childrenOf(pid)`，每层 `item { VersionNodeCard(node, depth, isRoot = false, onClick) }`。

节点卡片（私有 composable）：`Row` = `Spacer(width = (depth * 20).dp)` + 2dp 宽竖线 `Box(Modifier.width(2.dp).height(40.dp).background(OutlineVariantGray))`（depth>0 才画）+ `WanLiCard` 内含 `AuthorRow(nickname = node.authorNickname, subtitle = node.changeLog, trailing = { SocialBadge 或版本类型 chip })`；根节点卡片加 `SocialBadge("UP 主原版")`（芥末黄）或 `SocialBadge("${childCount} 个改编")`，改编版节点显示 `versionTypeLabel(node.versionType)`。

（`parentId`/`createdAt` 字段名以 `VersionTreeNodeDto` 实际为准；`onOpenVersion` 导航到 `Routes.recipeDetail(nodeId)`——若该路由已存在则复用。）

- [ ] **Step 2: 构建 + 测试 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest --tests "*VersionTree*"`
Expected: 全绿

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/recipe/VersionTreeScreen.kt
git commit -m "feat(version-tree): visual lineage tree with depth connectors and author avatars"
```

---

### Task 5: 厨房页视觉重做（功能不动）

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/kitchen/KitchenScreen.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/kitchen/DietaryScreen.kt`

**Stitch 基准：** `bea017fb61eb46c287183b596f4a32b9`。

- [ ] **Step 1: chip 表单 → 卡片分区**

`KitchenScreen`：现有「拥有食材/拥有工具」两个 `ChipSection` 的逻辑（增删 FilterChip、ViewModel 调用）不变，包装改为：

- 页面标题：自绘 header 换 `WanLiTopBar(title = "我的厨房")`
- 每个分区 = `WanLiCard { SectionHeader(title) + FlowRow(chips) }`；`🧂`→`Icons.Outlined.ShoppingBag`（食材），`🍳`→`Icons.Outlined.Kitchen`（工具），图标放在 `SectionHeader` 左侧（给 SectionHeader 加可选 `icon: ImageVector?` 参数）
- FilterChip 选中态自动继承新主题色

`DietaryScreen`：同样卡片化；`🛡️`→`Icons.Outlined.Shield`，分类图标 `🚫`→`Icons.Outlined.Block`、`💊`→`Icons.Outlined.MedicalServices`、`🙅`→`Icons.Outlined.BackHand`（若不可解析换 `Icons.Outlined.DoNotDisturb`），删除行的 `✕` 文本换 `Icons.Outlined.Close` IconButton。

- [ ] **Step 2: 构建 + 测试 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest --tests "*Kitchen*" --tests "*Dietary*"`
Expected: 全绿

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/kitchen
git commit -m "feat(kitchen): card-based sections with outlined icons, emoji removed"
```

---

### Task 6: 我的——身份区 + 做过照片墙

**Files:**
- Create: `app/src/main/kotlin/com/wanli/android/presentation/profile/ProfileViewModel.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/profile/ProfileScreen.kt`
- Test: `app/src/test/kotlin/com/wanli/android/presentation/profile/ProfileViewModelTest.kt`

**Stitch 基准：** `02bba32c51a445258bc243002ba47423`。

- [ ] **Step 1: 写失败测试**

```kotlin
package com.wanli.android.presentation.profile

// imports 同 HomeViewModelTest 的模式（mockk/runTest/ApiResult）

class ProfileViewModelTest {
    private val authRepository = mockk<AuthRepository>()
    private val socialRepository = mockk<SocialRepository>()

    @Test
    fun `load populates identity and photo wall`() = runTest {
        coEvery { authRepository.getMe() } returns ApiResult.Success(UserDto(id = "u1", email = "a@b.c", nickname = "小厨", avatarUrl = null))
        coEvery { socialRepository.getMyCookingRecords(any()) } returns ApiResult.Success(emptyList())

        val vm = ProfileViewModel(authRepository, socialRepository)

        val state = vm.uiState.value
        assertTrue(state is ProfileUiState.Success)
        assertEquals("小厨", (state as ProfileUiState.Success).nickname)
    }

    @Test
    fun `updateNickname refreshes identity`() = runTest {
        coEvery { authRepository.getMe() } returns ApiResult.Success(UserDto(id = "u1", email = "a@b.c", nickname = "旧", avatarUrl = null))
        coEvery { socialRepository.getMyCookingRecords(any()) } returns ApiResult.Success(emptyList())
        coEvery { authRepository.updateProfile("新昵称", null) } returns ApiResult.Success(UserDto(id = "u1", email = "a@b.c", nickname = "新昵称", avatarUrl = null))

        val vm = ProfileViewModel(authRepository, socialRepository)
        vm.updateNickname("新昵称")

        assertEquals("新昵称", (vm.uiState.value as ProfileUiState.Success).nickname)
    }
}
```

（`UserDto` 构造参数以实际声明为准。）

- [ ] **Step 2: 运行确认失败** `./gradlew :app:testDebugUnitTest --tests "*ProfileViewModelTest*"` → FAIL

- [ ] **Step 3: 实现 ProfileViewModel**

```kotlin
package com.wanli.android.presentation.profile

// imports 同 HomeViewModel 模式

sealed interface ProfileUiState {
    data object Loading : ProfileUiState
    data class Error(val message: String) : ProfileUiState
    data class Success(
        val nickname: String,
        val email: String,
        val avatarUrl: String?,
        val cookingRecords: List<MyCookingRecordDto>,
    ) : ProfileUiState
}

@HiltViewModel
class ProfileViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    private val socialRepository: SocialRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow<ProfileUiState>(ProfileUiState.Loading)
    val uiState: StateFlow<ProfileUiState> = _uiState.asStateFlow()

    init { load() }

    fun load() {
        viewModelScope.launch {
            _uiState.value = ProfileUiState.Loading
            val me = authRepository.getMe()
            if (me !is ApiResult.Success) {
                _uiState.value = ProfileUiState.Error((me as? ApiResult.Error)?.message ?: "加载失败")
                return@launch
            }
            val records = (socialRepository.getMyCookingRecords() as? ApiResult.Success)?.data ?: emptyList()
            _uiState.value = ProfileUiState.Success(
                nickname = me.data.nickname,
                email = me.data.email,
                avatarUrl = me.data.avatarUrl,
                cookingRecords = records,
            )
        }
    }

    fun updateNickname(nickname: String) {
        val current = _uiState.value as? ProfileUiState.Success ?: return
        viewModelScope.launch {
            when (val result = authRepository.updateProfile(nickname, current.avatarUrl)) {
                is ApiResult.Success -> _uiState.value = current.copy(nickname = result.data.nickname)
                else -> Unit // 失败提示后续以 snackbar 补充，本期静默保持旧值
            }
        }
    }
}
```

- [ ] **Step 4: 运行确认通过** → PASS

- [ ] **Step 5: 重写 ProfileScreen**

结构（未登录态保留现有引导逻辑，`👋`/`👨‍🍳` emoji 删除）：

```kotlin
LazyColumn(contentPadding = PaddingValues(20.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
    item { // 身份区
        Row(verticalAlignment = Alignment.CenterVertically) {
            UserAvatar(nickname = state.nickname, avatarUrl = state.avatarUrl, size = 64.dp)
            Spacer(Modifier.width(16.dp))
            Column(Modifier.weight(1f)) {
                Text(state.nickname, style = MaterialTheme.typography.headlineSmall)
                Text(state.email, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            IconButton(onClick = { showEditDialog = true }) { Icon(Icons.Outlined.Edit, "编辑资料") }
        }
    }
    if (photoRecords.isNotEmpty()) {
        item { SectionHeader("我的做过") }
        items(photoRecords.chunked(3)) { rowItems ->
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                rowItems.forEach { record ->
                    AsyncImage(
                        model = record.photoUrl ?: record.coverUrl,
                        contentDescription = record.recipeTitle,
                        modifier = Modifier.weight(1f).aspectRatio(1f).clip(MaterialTheme.shapes.small),
                        contentScale = ContentScale.Crop,
                    )
                }
                repeat(3 - rowItems.size) { Spacer(Modifier.weight(1f)) }
            }
        }
    }
    item { // 菜单
        WanLiCard {
            MenuRow(icon = Icons.Outlined.Restaurant, title = "我的适配版", onClick = onOpenAdaptations)
            MenuRow(icon = Icons.Outlined.RateReview, title = "待审核纠错", onClick = onOpenCorrections)
            MenuRow(icon = Icons.Outlined.Shield, title = "忌口管理", onClick = onOpenDietary)
        }
    }
    item { TextButton(onClick = onLogout, modifier = Modifier.fillMaxWidth()) { Text("退出登录", color = MaterialTheme.colorScheme.error) } }
}
```

`MenuRow` 改为图标版：`Row(Icon(icon, tint = primary) + Text(title) + Spacer(weight) + Icon(KeyboardArrowRight))`，整行 clickable，删除 emoji 参数。`showEditDialog` 为 `remember { mutableStateOf(false) }`，true 时弹 `AlertDialog`（一个 `OutlinedTextField` 昵称 + 确认调 `viewModel.updateNickname`）。

（照片墙取 `state.cookingRecords.filter { it.photoUrl != null || it.coverUrl != null }`；`onOpenAdaptations`/`onOpenCorrections`/`onOpenDietary`/`onLogout` 参数在 `MainScreen` 拼好传入。）

- [ ] **Step 6: 构建 + 全量测试 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest`
Expected: 全绿

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/profile app/src/test/kotlin/com/wanli/android/presentation/profile
git commit -m "feat(profile): identity header, cooked photo wall, settings menu with dietary entry"
```

---

### Task 7: 消息中心

**Files:**
- Create: `app/src/main/kotlin/com/wanli/android/presentation/messages/MessagesViewModel.kt`
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/messages/MessagesScreen.kt`（替换地基计划的占位实现）
- Test: `app/src/test/kotlin/com/wanli/android/presentation/messages/MessagesViewModelTest.kt`

**Stitch 基准：** `4e7d0f7bfb564b3eba162d3bbab725ad`。

- [ ] **Step 1: 写失败测试**

仿照 ProfileViewModelTest：`getMessages` 返回 2 条 → Success 且 size 2；Error → Error 状态。

- [ ] **Step 2: 实现 MessagesViewModel**

```kotlin
package com.wanli.android.presentation.messages

// imports 同 HomeViewModel 模式

sealed interface MessagesUiState {
    data object Loading : MessagesUiState
    data class Error(val message: String) : MessagesUiState
    data class Success(val items: List<MessageDto>) : MessagesUiState
}

@HiltViewModel
class MessagesViewModel @Inject constructor(
    private val socialRepository: SocialRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow<MessagesUiState>(MessagesUiState.Loading)
    val uiState: StateFlow<MessagesUiState> = _uiState.asStateFlow()

    init { refresh() }

    fun refresh() {
        viewModelScope.launch {
            _uiState.value = MessagesUiState.Loading
            when (val result = socialRepository.getMessages()) {
                is ApiResult.Success -> _uiState.value = MessagesUiState.Success(result.data)
                is ApiResult.Error -> _uiState.value = MessagesUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }
}
```

- [ ] **Step 3: 实现 MessagesScreen 列表**

每行：`WanLiCard { AuthorRow(nickname = 标题文案, subtitle = item.text, trailing = 时间) }`，左侧在 AuthorRow 前再放类型图标：

```kotlin
val (icon, tint) = when (item.type) {
    "comment" -> Icons.Outlined.ChatBubbleOutline to MaterialTheme.colorScheme.primary
    "cooking_record" -> Icons.Outlined.Restaurant to MaterialTheme.colorScheme.primary
    "correction_reviewed" -> Icons.Outlined.CheckCircle to SuccessGreen40
    "correction_pending" -> Icons.Outlined.RateReview to WarningAmber40
    else -> Icons.Outlined.Notifications to MaterialTheme.colorScheme.primary
}
```

标题文案：`comment` → `"${item.actorNickname} 评论了《${item.recipeTitle}》"`；`cooking_record` → `"${item.actorNickname} 做了《${item.recipeTitle}》"`；`correction_reviewed` → `"《${item.recipeTitle}》纠错审核结果"`；`correction_pending` → `"${item.actorNickname} 提交了《${item.recipeTitle}》的纠错"`。`cooking_record` 行在 subtitle 前加 `RatingStars(item.rating)`。整卡 `onClick`：`item.recipeVersionId?.let(onOpenRecipe)`。空态 `EmptyState(Icons.Outlined.Notifications, "暂无消息", "有人评论或做过你的菜谱时会出现在这里")`。

- [ ] **Step 4: 构建 + 测试 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest --tests "*Messages*"`
Expected: 全绿

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/messages app/src/test/kotlin/com/wanli/android/presentation/messages
git commit -m "feat(messages): aggregated message center with typed rows"
```

---

### Task 8: 登录页品牌化

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/auth/LoginScreen.kt`

**Stitch 基准：** `b03e2c67658c43e68551aa65e4264dca`。

- [ ] **Step 1: 重排布局**

现有 `LoginViewModel` 与表单逻辑（邮箱、验证码、倒计时、提交）不动，布局改为：

1. 顶部品牌区（页面上 1/3 垂直居中）：`Box(96.dp, CircleShape, SageSoft)` 内 `Icon(Icons.Outlined.SoupKitchen, 48.dp, tint = primary)`（替换 `🥣`）→ `Text("碗里", headlineMedium 衬线)` → `Text("把视频变成能下锅的菜谱", bodyMedium, onSurfaceVariant)`
2. `WanLiCard` 包裹邮箱输入框 + 验证码行（输入框 + 倒计时 TextButton）
3. 主按钮「登录 / 注册」：满宽 `Button(shape = MaterialTheme.shapes.large)`（胶囊感）
4. 底部 `Text("登录即表示同意用户协议与隐私政策", bodySmall, onSurfaceVariant, Center)`

- [ ] **Step 2: 构建 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug`
Expected: BUILD SUCCESSFUL

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/auth/LoginScreen.kt
git commit -m "feat(login): branded login with sage identity block and pill CTA"
```

---

### Task 9: 适配我的厨房·结果页

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/recipe/AdaptRecipeScreen.kt`

**Stitch 基准：** `d522585cc07742b3b17c4117d419ed8c`。

- [ ] **Step 1: 结果区分区重做**

现有 ViewModel 触发适配、拿到 `AdaptationNotesDto` 的逻辑不动（`missingIngredients`/`missingTools`/`substitutions`/`flavorChanges` 字段名以 DTO 为准）。结果区改为：

1. 成功横幅：`Surface(color = SuccessGreen90, shape = medium)` 内 `Row(Icon(CheckCircle, tint = SuccessGreen40) + Column("已根据你的厨房完成适配" + "N 处替换 · M 种缺少 · K 条口味建议" 灰字))`（替换 `🧑‍🍳` emoji）
2. `SectionHeader("替换")` + 每项 `WanLiCard`：`Row(Icon(SwapHoriz, primary) + Column)`，Column 内 `Row(Text(original, style = bodyMedium.copy(textDecoration = TextDecoration.LineThrough), color = onSurfaceVariant) + Icon(ArrowForward, 16dp) + Text(replacement, color = primary, fontWeight = SemiBold))` + reason 灰字
3. `SectionHeader("缺少")` + `WanLiCard` 列表：`Icon(WarningAmber, WarningAmber40) + Text(name)` + 右侧 `Text("可省略", bodySmall)`
4. `SectionHeader("口味变化")` + `Surface(color = MustardSoft)` 卡：`Icon(Lightbulb, MustardDeep) + Text(flavorChanges)`
5. 底部固定条（`Scaffold.bottomBar`）：`Button("保存为我的版本")` + `OutlinedButton("查看原步骤")`——按钮行为沿用现有逻辑；若现有页面没有「保存」动作（适配已自动落库），主按钮文案改「完成」并 popBackStack

注意：Stitch 稿中的过敏原警示条本期不实现（后端未返回过敏原维度），食物大图省略（静态稿瑕疵）。

- [ ] **Step 2: 构建 + 测试 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest --tests "*AdaptRecipe*"`
Expected: 全绿

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/recipe/AdaptRecipeScreen.kt
git commit -m "feat(adapt): sectioned adaptation result with substitution/missing/flavor cards"
```

---

### Task 10: 校对菜谱编辑页

**Files:**
- Modify: `app/src/main/kotlin/com/wanli/android/presentation/recipe/ProofreadScreen.kt`

**Stitch 基准：** `b13eb6ba6f0c435ca1544344028a6d0e`。

- [ ] **Step 1: 编辑列表视觉重做**

现有 `ProofreadViewModel` 的编辑/保存/发布逻辑不动，UI 改为：

1. 顶部摘要卡：`WanLiCard { Row(封面 AsyncImage 64.dp 圆角 + Column(菜名 OutlinedTextField + "来源：B 站 · ${upNickname}" 灰字)) }`
2. `SectionHeader("食材")` + `WanLiCard`：每行 `Icon(DragIndicator, outline) + 食材编辑行 + IconButton(Edit)`；`ingredient.isExplicit == false` 的行在行尾加 `SocialBadge("AI 推测")`；卡片底部 `OutlinedButton("+ 添加食材", fillMaxWidth)`（虚线样式无原生支持，用描边按钮代替）
3. `SectionHeader("步骤")` + `WanLiCard`：每行 `Box(24.dp 圆形 Sage90 底 + Text(序号, primary)) + 步骤编辑行 + 时间戳 chip（`videoTimestamp` 非空时显示 `mm:ss`）`
4. 底部固定条：`Button("发布到社区")` + `OutlinedButton("保存草稿")`（沿用现有 publish/save 调用）

- [ ] **Step 2: 构建 + 测试 + 提交检查点（可选）**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest --tests "*Proofread*"`
Expected: 全绿

```bash
git add app/src/main/kotlin/com/wanli/android/presentation/recipe/ProofreadScreen.kt
git commit -m "feat(proofread): structured editor with AI-inferred badges and bottom action bar"
```

---

### Task 11: 全局收尾

**Files:** 全项目扫描

- [ ] **Step 1: emoji 清零审计**

Run: `grep -rnP '[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}★☆✕🔗🔒]' app/src/main/kotlin/ || echo "CLEAN"`
Expected: `CLEAN`；残留逐个替换为图标/删除

- [ ] **Step 2: 旧色值清零**

Run: `grep -rn "Terracotta\|C8441C" app/src/ || echo "CLEAN"`
Expected: `CLEAN`

- [ ] **Step 3: 全量验证**

Run: `./gradlew :app:assembleDebug && ./gradlew :app:testDebugUnitTest`
Expected: BUILD SUCCESSFUL + 全部 PASS

- [ ] **Step 4: 人工走查清单（模拟器/真机）**

对照 Stitch 截图逐屏核对：首页 Feed（含空态）→ 发现瀑布流 → 详情（做过墙/谱系芯片）→ 版本树 → 厨房 → 消息 → 我的（照片墙/编辑昵称）→ 登录 → 适配结果 → 校对。记录偏差点并修复。

- [ ] **Step 5: 提交检查点（可选）**

```bash
git add -A
git commit -m "chore: final redesign sweep (emoji purge, token cleanup)"
```

---

## Self-Review 记录

- 规格覆盖：§3 五个 Tab 对应 Task 1/2/5/6/7；§5 社交改造四项（首页 Feed、详情、版本树、我的）对应 Task 1/3/4/6；次级屏（登录/适配/校对/空态）对应 Task 8/9/10 + 各屏 EmptyState；§7 导出基准即 Stitch 截图对照走查（Task 11 Step 4）。
- 刻意省略（与设计文档非目标一致）：点赞/关注/私信、暗黑模式、大字烹饪模式、消息已读、过敏原警示条。
- 类型一致性：`HomeFeedUiState`/`ProfileUiState`/`MessagesUiState` 的构造在各任务内自洽；共享组件签名以地基计划 Task 3 为准。

---

## 执行记录

### Task 2（发现页瀑布流）— 已完成并过评审
- 落地与计划一致；DTO 字段 snake_case（仅 cookingRecordCount 为 camelCase+@SerialName）。
- 质量修复：错误分支改为 weight(1f) + 重试按钮（原 fillMaxSize 会把按钮挤出可视区）；selectedTab 改从 uiState 派生；封面 contentDescription=null；DiscoverScreen 增加 modifier 参数。
- 评审驳回项：LazyVerticalStaggeredGrid 需 @OptIn(ExperimentalFoundationApi) 的 claim 被构建成功证伪，未添加。
- Follow-up：
  1. HomeScreen FeedCard 封面同样 contentDescription=title 重复播报，T11 清扫时统一改 null。
  2. DiscoverViewModel 快速切 tab 时 load() 协程竞态（last-completed-wins），后续可加 collectLatest/取消。
  3. 旧发现卡片的「原视频已失效」标识已随卡片重做移除，仅详情页保留失效提示。

### Task 3（详情页社交化）— 已完成并过评审
- 落地与计划一致；snake_case DTO（recipe_id/version_type/photo_url/created_at）；created_at 为 ISO 字符串 → 副标题 take(10)；无头像字段 → AuthorRow 首字母 fallback。
- 规格补漏：CommunitySection 错误态补 EmptyState。
- 质量修复：谱系 chip heightIn(48dp)、封面 contentDescription=null + isNullOrBlank 守卫、canDerive 去重、formatTimestamp Locale.US、做过墙照片加 contentDescription、photoUri rememberSaveable、社区 loading 居中。
- Follow-up：CommunityViewModel.submitComment fire-and-forget，评论框提交即清空、失败丢文案 —— 需要 VM 暴露成功态才能改，记入后续。

### Task 4（版本谱系树）— 已完成并过评审
- 落地与计划一致；snake_case DTO；导航复用现有 Routes.recipeDetail；versionTypeLabel 复用 RecipeDetailScreen（原本即非 private，非本任务改动）。
- 接受的偏差：节点卡片含版本标题 Text（无标题卡片信息不足，控制者批准）。
- 质量修复：childrenByParent remember 索引（原每重组 O(n²)）；深度无障碍语义（UP 主原版/第 N 代）；全孤儿数据回退空态；缩进上限 minOf(depth,4)；标题 maxLines=2；ISO 排序假设注释。

### Task 5（厨房/忌口卡片化）— 已完成并过评审
- 落地与计划一致；SectionHeader 新增可选 icon 参数（向后兼容，既有调用点均为命名参数/仅 title）；功能逻辑不变；无 Kitchen/Dietary 测试类，全量套件验证。
- 质量修复：删除按钮 contentDescription 带食材名；删除多余 Spacer（16dp 节奏统一）；TagStyle Color 导入化。
- 接受项：加载态保留 LoadingIndicator（功能不动原则），统一留待 T11 清扫；忌口警示 banner 保持 errorContainer Card 而非 WanLiCard（语义正确）。
- 备注：ProofreadScreen 使用自带私有 SectionHeader(onAdd=...)（与共享组件同名不同签名，编译无冲突），T10 时核对。

### Task 6（我的页）— 已完成并过评审
- TDD 落地；UserDto.avatar_url snake_case → VM 映射 avatarUrl；ProfileScreen 签名 email→isLoggedIn（email 入 VM state）；未登录分支用共享 EmptyState；昵称空 fallback「碗里的厨子」。
- 质量修复：LoggedInProfile 拆分（未登录不再触发 getMe）；updateNickname 成功分支重读 state 防旧值覆写；照片墙行 key=row.first().id；新增测试锁定「记录拉取失败静默空列表」契约（44 tests）。
- Follow-up：MainViewModel.email 现已无消费方，T11 清扫时评估删除。

### Task 7（消息中心）— 已完成并过评审
- TDD 落地；MessageDto camelCase（createdAt @SerialName("created_at") RFC3339 → take(10)）；类型字符串与后端 message_service.go 逐一核对一致。
- 接受的偏差：AuthorRow.subtitle 为 String?，cooking_record 的 RatingStars 移到 AuthorRow 下方独立行（视觉等价）；messageIcon 为 @Composable。
- 质量修复：消息类型字符串提为 private const；昵称空白 fallback「有人」；RFC3339 假设注释。

### Task 8（登录页品牌化）— 已完成并过评审
- 落地与计划一致；现有代码无倒计时状态（LoginViewModel 仅 Idle/Loading/CodeSent/LoggedIn/Error），保留纯发送验证码按钮；签名不变。
- 质量修复：外层 Column 加 verticalScroll + imePadding（原 weight(1f) 品牌区在键盘弹出时会把表单挤出且 weight 在可滚动 Column 中非法）；品牌区改固定 280.dp 高；陈旧注释更新；24.dp 根内边距判定为登录页刻意选择保留。

### Task 9（适配结果页）— 已完成并过评审
- 落地与计划一致；DTO snake_case（missing_ingredients/missing_tools/flavor_changes List<String>，substitutions 含 original/replacement/reason）；缺少区合并食材+工具。
- 接受的偏差：「完成」按钮 → onOpenResult(detail.id)（适配自动私有落库，popBackStack 会丢掉唯一入口）；Ready 态 🧑‍🍳 → DinnerDining 图标；加载态保留 CircularProgressIndicator（生成中而非内容加载）。
- 质量修复：bottomBar 加 navigationBarsPadding（edge-to-edge）；替换行加合并语义「X，已替换为 Y。原因：Z」；全零统计回退文案；替换行改两行结构防溢出；成功横幅改为复用 StatusBanner(SUCCESS)（标题深色文字刻意分歧已注释）。

### Task 10（校对编辑页）— 已完成并过评审
- 落地与计划一致；私有 SectionHeader(onAdd) 删除改用共享组件；DTO snake_case（cover_url/up_nickname/is_explicit/video_timestamp，form model 为 camelCase）。
- 接受的适配：ProofreadViewModel 增量改动（Editing 加 coverUrl/upNickname，逻辑不变）；Edit 按钮省略（行内编辑已是事实）；时间戳改纯展示 chip（编辑能力保留在 form model）；补「+ 添加步骤」按钮（对称 + 防删后无法新增）；分量/耗时/小贴士保留在摘要卡。
- 质量修复（4 Important + 5 Minor）：DragIndicator 改装饰性；hint 分 isError 着色（原错误显示成功色）；save() 不再整体替换表单（保留输入中编辑与焦点）；食材行改两行结构防窄屏溢出；按钮文案静态化；consumePublished 防配置变更重复导航；行 key(index)；删除按钮 contentDescription 带身份；formatTimestamp 负数钳制。新增 3 个测试（共 50 tests 全绿）。
- **重要发现留给 T11**：MyAdaptationsScreen.kt 与 PendingCorrectionsScreen.kt 仍有 emoji（2026-07-24 的文件，此前任务均未覆盖）。
- Follow-up：拖拽排序图标为纯装饰（无重排逻辑），如要真拖拽需后续迭代；videoTimestamp 无编辑 UI（仅展示）。

### Task 11（全局收尾）— 已完成并验证
- emoji 审计：PendingCorrectionsScreen ✅ / MyAdaptationsScreen 🥡 清除，两屏加载/错误/空态对齐共享组件（SkeletonCard/EmptyState+重试）。全项目 grep CLEAN。
- 旧色值审计：Terracotta/C8441C 无残留；无 theme/Color.kt 外的硬编码 Color(0x...)。
- Follow-up 处理：HomeScreen 封面 contentDescription=null；MainViewModel.email 删除（含 map import 清理）；Kitchen/Dietary 加载态换 SkeletonCard、LoadingIndicator 组件删除；忌口警示 banner 换 StatusBanner(WARNING)（警示非失败，不用 ERROR）。
- 评审备注：Kitchen/Dietary 错误态在 verticalScroll Column 内显示为顶部流式而非居中（fillMaxSize 在无界约束下退化为 wrap，非崩溃），可接受的视觉细节。
- 全部 11 个界面任务完成，50 tests 全绿。

---

## 最终评审与收尾（2026-07-30）

- 全实现终审：READY。双端测试绿（Android 55 tests / Go 全 ok），API 契约逐字段对齐，导航无死路由，emoji/旧色值 grep CLEAN。
- 终审后修复：AuthRepository.login() 网络异常崩溃（try/catch → ApiResult.Error + 3 个新测试）；删除未用 token SageBrand/CreamSurfaceLow；api-contract.md GET /me 示例补 nickname/avatar_url。
- 仍开放的 follow-up（已记录、v1 接受）：
  1. 后端：cooking_records 部分索引迁移；ListPublicRecipes COUNT 包装优化；ListPendingByCreator 无上限；消息无已读态。
  2. wrap helper 三处复制（待第 4 处出现时抽取，顺带处理 CancellationException 吞掉与 success=false 丢 message）。
  3. DiscoverViewModel 切 tab 竞态；CommunityViewModel.submitComment fire-and-forget。
  4. RecipeDetail/RecipeEdit 全页加载仍是 CircularProgressIndicator（可换 SkeletonCard 统一）；错误态两式并存（ErrorMessage+按钮 vs EmptyState+重试）。
  5. Proofread 拖拽柄为装饰；videoTimestamp 纯展示。
