package com.wanli.android.presentation.main

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.Explore
import androidx.compose.material.icons.outlined.Home
import androidx.compose.material.icons.outlined.Kitchen
import androidx.compose.material.icons.outlined.Lock
import androidx.compose.material.icons.outlined.Notifications
import androidx.compose.material.icons.outlined.Person
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.NavigationBar
import androidx.compose.material3.NavigationBarItem
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.vector.ImageVector
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
import com.wanli.android.presentation.discover.DiscoverScreen
import com.wanli.android.presentation.kitchen.KitchenScreen
import com.wanli.android.presentation.home.HomeScreen
import com.wanli.android.presentation.messages.MessagesScreen
import com.wanli.android.presentation.profile.ProfileScreen

sealed class MainTab(val route: String, val title: String, val icon: ImageVector) {
    data object Home : MainTab("home", "首页", Icons.Outlined.Home)
    data object Discover : MainTab("discover", "发现", Icons.Outlined.Explore)
    data object Kitchen : MainTab("kitchen", "厨房", Icons.Outlined.Kitchen)
    data object Messages : MainTab("messages", "消息", Icons.Outlined.Notifications)
    data object Profile : MainTab("profile", "我的", Icons.Outlined.Person)
}

@Composable
fun MainScreen(
    onNavigateToLogin: () -> Unit,
    onOpenRecipe: (versionId: String) -> Unit = {},
    onProofread: (versionId: String) -> Unit = {},
    onOpenPendingCorrections: () -> Unit = {},
    onOpenMyAdaptations: () -> Unit = {},
    onOpenDietary: () -> Unit = {},
    navController: NavHostController = rememberNavController(),
    viewModel: MainViewModel = hiltViewModel()
) {
    val isLoggedIn by viewModel.isLoggedIn.collectAsStateWithLifecycle(initialValue = false)
    val tabs = listOf(MainTab.Home, MainTab.Discover, MainTab.Kitchen, MainTab.Messages, MainTab.Profile)

    Scaffold(
        bottomBar = {
            NavigationBar {
                val navBackStackEntry by navController.currentBackStackEntryAsState()
                val currentDestination = navBackStackEntry?.destination

                tabs.forEach { tab ->
                    NavigationBarItem(
                        icon = { Icon(tab.icon, contentDescription = tab.title) },
                        label = { Text(tab.title) },
                        selected = currentDestination?.hierarchy?.any { it.route == tab.route } == true,
                        onClick = {
                            navController.navigate(tab.route) {
                                popUpTo(navController.graph.findStartDestination().id) {
                                    saveState = true
                                }
                                launchSingleTop = true
                                restoreState = true
                            }
                        }
                    )
                }
            }
        }
    ) { padding ->
        val modifier = Modifier.padding(padding)

        NavHost(
            navController = navController,
            startDestination = MainTab.Home.route,
            modifier = modifier
        ) {
            composable(MainTab.Home.route) {
                HomeScreen(
                    onNavigateToLogin = onNavigateToLogin,
                    onOpenRecipe = onOpenRecipe,
                    onProofread = onProofread,
                    onOpenMessages = {
                        navController.navigate(MainTab.Messages.route) {
                            popUpTo(navController.graph.findStartDestination().id) {
                                saveState = true
                            }
                            launchSingleTop = true
                            restoreState = true
                        }
                    },
                    onOpenDiscover = {
                        navController.navigate(MainTab.Discover.route) {
                            popUpTo(navController.graph.findStartDestination().id) {
                                saveState = true
                            }
                            launchSingleTop = true
                            restoreState = true
                        }
                    }
                )
            }
            composable(MainTab.Discover.route) {
                DiscoverScreen(onOpenRecipe = onOpenRecipe)
            }
            composable(MainTab.Kitchen.route) {
                if (isLoggedIn) {
                    KitchenScreen()
                } else {
                    LoginRequired(onLogin = onNavigateToLogin)
                }
            }
            composable(MainTab.Messages.route) {
                if (isLoggedIn) {
                    MessagesScreen(onOpenRecipe = onOpenRecipe)
                } else {
                    LoginRequired(onLogin = onNavigateToLogin)
                }
            }
            composable(MainTab.Profile.route) {
                ProfileScreen(
                    isLoggedIn = isLoggedIn,
                    onLogout = { viewModel.logout() },
                    onLogin = onNavigateToLogin,
                    onOpenPendingCorrections = onOpenPendingCorrections,
                    onOpenMyAdaptations = onOpenMyAdaptations,
                    onOpenDietary = onOpenDietary
                )
            }
        }
    }
}

@Composable
private fun LoginRequired(
    onLogin: () -> Unit,
    modifier: Modifier = Modifier
) {
    Column(
        modifier = modifier.fillMaxSize().padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Icon(
            imageVector = Icons.Outlined.Lock,
            contentDescription = null,
            tint = MaterialTheme.colorScheme.primary,
            modifier = Modifier.size(48.dp)
        )
        Text(
            text = "需要登录后查看",
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(top = 12.dp)
        )
        Button(onClick = onLogin, modifier = Modifier.padding(top = 16.dp)) {
            Text("去登录")
        }
    }
}
