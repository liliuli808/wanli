package com.wanli.android.presentation.main

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Explore
import androidx.compose.material.icons.filled.Home
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.outlined.Explore
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.Lock
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material3.Button
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalClipboardManager
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.NavDestination.Companion.hierarchy
import androidx.navigation.NavGraph.Companion.findStartDestination
import androidx.navigation.NavHostController
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.compose.currentBackStackEntryAsState
import androidx.navigation.compose.rememberNavController
import com.wanli.android.presentation.MainViewModel
import com.wanli.android.presentation.components.AppBottomBar
import com.wanli.android.presentation.components.BottomTab
import com.wanli.android.presentation.components.WanLiTopBar
import com.wanli.android.presentation.discover.DiscoverScreen
import com.wanli.android.presentation.home.HomeScreen
import com.wanli.android.presentation.home.ImportViewModel
import com.wanli.android.presentation.kitchen.KitchenScreen
import com.wanli.android.presentation.messages.MessagesScreen
import com.wanli.android.presentation.profile.ProfileScreen
import com.wanli.android.presentation.theme.CardWhite

object MainRoutes {
    const val HOME = "home"
    const val DISCOVER = "discover"
    const val MESSAGES = "messages"
    const val PROFILE = "profile"
    const val KITCHEN = "kitchen"
}

private val mainTabs = listOf(
    BottomTab(MainRoutes.HOME, "首页", Icons.Outlined.Home, Icons.Filled.Home),
    BottomTab(MainRoutes.DISCOVER, "发现", Icons.Outlined.Explore, Icons.Filled.Explore),
    BottomTab(MainRoutes.MESSAGES, "消息", Icons.Outlined.Notifications, Icons.Filled.Notifications),
    BottomTab(MainRoutes.PROFILE, "我的", Icons.Outlined.Person, Icons.Filled.Person),
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun MainScreen(
    onNavigateToLogin: () -> Unit,
    onOpenRecipe: (versionId: String) -> Unit = {},
    onProofread: (versionId: String) -> Unit = {},
    onOpenPendingCorrections: () -> Unit = {},
    onOpenMyAdaptations: () -> Unit = {},
    onOpenDietary: () -> Unit = {},
    navController: NavHostController = rememberNavController(),
    viewModel: MainViewModel = hiltViewModel(),
    importViewModel: ImportViewModel = hiltViewModel(),
) {
    val isLoggedIn by viewModel.isLoggedIn.collectAsStateWithLifecycle(initialValue = false)
    val importState by importViewModel.uiState.collectAsStateWithLifecycle()
    var showImportSheet by rememberSaveable { mutableStateOf(false) }
    var urlInput by rememberSaveable { mutableStateOf("") }
    val clipboard = LocalClipboardManager.current
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    fun goTab(route: String) {
        navController.navigate(route) {
            popUpTo(navController.graph.findStartDestination().id) { saveState = true }
            launchSingleTop = true
            restoreState = true
        }
    }

    Scaffold(
        containerColor = MaterialTheme.colorScheme.background,
        bottomBar = {
            val navBackStackEntry by navController.currentBackStackEntryAsState()
            val currentRoute = navBackStackEntry?.destination?.route
            AppBottomBar(
                tabs = mainTabs,
                selectedRoute = currentRoute,
                onSelect = { goTab(it.route) },
                onCenterClick = {
                    importViewModel.reset()
                    showImportSheet = true
                },
            )
        },
    ) { padding ->
        NavHost(
            navController = navController,
            startDestination = MainRoutes.HOME,
            modifier = Modifier.padding(padding).fillMaxSize(),
        ) {
            composable(MainRoutes.HOME) {
                HomeScreen(
                    onNavigateToLogin = onNavigateToLogin,
                    onOpenRecipe = onOpenRecipe,
                    onProofread = onProofread,
                    onOpenMessages = { goTab(MainRoutes.MESSAGES) },
                    onOpenDiscover = { goTab(MainRoutes.DISCOVER) },
                    onOpenImport = {
                        importViewModel.reset()
                        showImportSheet = true
                    },
                )
            }
            composable(MainRoutes.DISCOVER) {
                DiscoverScreen(onOpenRecipe = onOpenRecipe)
            }
            composable(MainRoutes.MESSAGES) {
                if (isLoggedIn) {
                    MessagesScreen(onOpenRecipe = onOpenRecipe)
                } else {
                    LoginRequired(onLogin = onNavigateToLogin, title = "消息")
                }
            }
            composable(MainRoutes.PROFILE) {
                ProfileScreen(
                    isLoggedIn = isLoggedIn,
                    onLogout = { viewModel.logout() },
                    onLogin = onNavigateToLogin,
                    onOpenPendingCorrections = onOpenPendingCorrections,
                    onOpenMyAdaptations = onOpenMyAdaptations,
                    onOpenDietary = onOpenDietary,
                    onOpenKitchen = { navController.navigate(MainRoutes.KITCHEN) },
                )
            }
            composable(MainRoutes.KITCHEN) {
                if (isLoggedIn) {
                    KitchenScreen(onBack = { navController.popBackStack() })
                } else {
                    LoginRequired(onLogin = onNavigateToLogin, title = "我的厨房")
                }
            }
        }
    }

    if (showImportSheet) {
        ModalBottomSheet(
            onDismissRequest = {
                showImportSheet = false
                importViewModel.reset()
            },
            sheetState = sheetState,
            containerColor = CardWhite,
            shape = RoundedCornerShape(topStart = 28.dp, topEnd = 28.dp),
        ) {
            ImportSheetContent(
                url = urlInput,
                onUrlChange = { urlInput = it },
                importState = importState,
                onSubmit = { importViewModel.submit(urlInput) },
                onPaste = {
                    clipboard.getText()?.text?.let { urlInput = it.trim() }
                },
                onOpenRecipe = { id ->
                    showImportSheet = false
                    onOpenRecipe(id)
                },
                onProofread = { id ->
                    showImportSheet = false
                    importViewModel.reset()
                    onProofread(id)
                },
                onNavigateToLogin = {
                    showImportSheet = false
                    onNavigateToLogin()
                },
                onReset = {
                    importViewModel.reset()
                    urlInput = ""
                },
            )
        }
    }
}

@Composable
private fun LoginRequired(
    onLogin: () -> Unit,
    modifier: Modifier = Modifier,
    title: String? = null,
) {
    Column(modifier = modifier.fillMaxSize()) {
        if (title != null) {
            WanLiTopBar(title = title)
        }
        Column(
            modifier = Modifier.weight(1f).fillMaxWidth().padding(32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center,
        ) {
            Icon(
                imageVector = Icons.Outlined.Lock,
                contentDescription = null,
                tint = MaterialTheme.colorScheme.primary,
                modifier = Modifier.size(48.dp),
            )
            Text(
                text = "需要登录后查看",
                style = MaterialTheme.typography.titleMedium,
                modifier = Modifier.padding(top = 12.dp),
            )
            Button(onClick = onLogin, modifier = Modifier.padding(top = 16.dp)) {
                Text("去登录")
            }
        }
    }
}
