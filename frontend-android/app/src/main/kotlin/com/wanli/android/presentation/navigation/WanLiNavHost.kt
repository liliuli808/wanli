package com.wanli.android.presentation.navigation

import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.navigation.NavHostController
import androidx.navigation.NavType
import androidx.navigation.compose.NavHost
import androidx.navigation.compose.composable
import androidx.navigation.navArgument
import com.wanli.android.presentation.auth.LoginScreen
import com.wanli.android.presentation.kitchen.DietaryScreen
import com.wanli.android.presentation.main.MainScreen
import com.wanli.android.presentation.profile.BilibiliAuthScreen
import com.wanli.android.presentation.recipe.ProofreadScreen
import com.wanli.android.presentation.recipe.EDIT_MODE_CORRECTION
import com.wanli.android.presentation.recipe.EDIT_MODE_FORK
import com.wanli.android.presentation.recipe.PendingCorrectionsScreen
import com.wanli.android.presentation.recipe.AdaptRecipeScreen
import com.wanli.android.presentation.recipe.MyAdaptationsScreen
import com.wanli.android.presentation.recipe.MyDraftsScreen
import com.wanli.android.presentation.recipe.RecipeEditScreen
import com.wanli.android.presentation.recipe.RecipeDetailScreen
import com.wanli.android.presentation.recipe.VersionTreeScreen

object Routes {
    const val MAIN = "main"
    const val LOGIN = "login"
    const val RECIPE_DETAIL = "recipe-version/{versionId}"
    const val PROOFREAD = "proofread/{versionId}"
    const val RECIPE_EDIT = "recipe-version/{versionId}/edit/{editMode}"
    const val VERSION_TREE = "recipe/{recipeId}/version-tree"
    const val PENDING_CORRECTIONS = "correction-proposals/pending"
    const val ADAPT_RECIPE = "recipe-version/{versionId}/adapt"
    const val MY_ADAPTATIONS = "me/adaptations"
    const val MY_DRAFTS = "me/recipe-drafts"
    const val DIETARY = "me/dietary-restrictions"
    const val BILIBILI_AUTH = "me/bilibili-credential"

    fun recipeDetail(versionId: String) = "recipe-version/$versionId"
    fun proofread(versionId: String) = "proofread/$versionId"
    fun recipeEdit(versionId: String, editMode: String) = "recipe-version/$versionId/edit/$editMode"
    fun versionTree(recipeId: String) = "recipe/$recipeId/version-tree"
    fun adaptRecipe(versionId: String) = "recipe-version/$versionId/adapt"
}

@Composable
fun WanLiNavHost(
    navController: NavHostController,
    modifier: Modifier = Modifier,
    onCheckForUpdate: () -> Unit = {},
) {
    NavHost(
        navController = navController,
        startDestination = Routes.MAIN,
        modifier = modifier
    ) {
        composable(Routes.MAIN) {
            MainScreen(
                onNavigateToLogin = { navController.navigate(Routes.LOGIN) },
                onOpenRecipe = { versionId -> navController.navigate(Routes.recipeDetail(versionId)) },
                onProofread = { versionId -> navController.navigate(Routes.proofread(versionId)) },
                onOpenPendingCorrections = { navController.navigate(Routes.PENDING_CORRECTIONS) },
                onOpenMyAdaptations = { navController.navigate(Routes.MY_ADAPTATIONS) },
                onOpenMyDrafts = { navController.navigate(Routes.MY_DRAFTS) },
                onOpenDietary = { navController.navigate(Routes.DIETARY) },
                onOpenBilibiliAuth = { navController.navigate(Routes.BILIBILI_AUTH) },
                onCheckForUpdate = onCheckForUpdate,
            )
        }
        composable(Routes.LOGIN) {
            LoginScreen(
                onLoggedIn = {
                    navController.popBackStack()
                }
            )
        }
        composable(
            route = Routes.RECIPE_DETAIL,
            arguments = listOf(navArgument("versionId") { type = NavType.StringType })
        ) {
            RecipeDetailScreen(
                onBack = { navController.popBackStack() },
                onCorrection = { versionId ->
                    navController.navigate(Routes.recipeEdit(versionId, EDIT_MODE_CORRECTION))
                },
                onFork = { versionId ->
                    navController.navigate(Routes.recipeEdit(versionId, EDIT_MODE_FORK))
                },
                onAdapt = { versionId ->
                    navController.navigate(Routes.adaptRecipe(versionId))
                },
                onVersionTree = { recipeId -> navController.navigate(Routes.versionTree(recipeId)) }
            )
        }
        composable(
            route = Routes.RECIPE_EDIT,
            arguments = listOf(
                navArgument("versionId") { type = NavType.StringType },
                navArgument("editMode") { type = NavType.StringType }
            )
        ) {
            RecipeEditScreen(
                onBack = { navController.popBackStack() },
                onSubmitted = { newVersionId ->
                    if (newVersionId == null) {
                        navController.popBackStack()
                    } else {
                        navController.navigate(Routes.recipeDetail(newVersionId)) {
                            popUpTo(Routes.MAIN)
                        }
                    }
                }
            )
        }
        composable(
            route = Routes.VERSION_TREE,
            arguments = listOf(navArgument("recipeId") { type = NavType.StringType })
        ) {
            VersionTreeScreen(
                onBack = { navController.popBackStack() },
                onOpenVersion = { versionId -> navController.navigate(Routes.recipeDetail(versionId)) }
            )
        }
        composable(Routes.PENDING_CORRECTIONS) {
            PendingCorrectionsScreen(onBack = { navController.popBackStack() })
        }
        composable(
            route = Routes.ADAPT_RECIPE,
            arguments = listOf(navArgument("versionId") { type = NavType.StringType })
        ) {
            AdaptRecipeScreen(
                onBack = { navController.popBackStack() },
                onOpenResult = { versionId ->
                    navController.navigate(Routes.recipeDetail(versionId)) {
                        popUpTo(Routes.MAIN)
                    }
                }
            )
        }
        composable(Routes.MY_ADAPTATIONS) {
            MyAdaptationsScreen(
                onBack = { navController.popBackStack() },
                onOpenVersion = { versionId ->
                    navController.navigate(Routes.recipeDetail(versionId))
                }
            )
        }
        composable(Routes.MY_DRAFTS) {
            MyDraftsScreen(
                onBack = { navController.popBackStack() },
                onOpenProofread = { versionId -> navController.navigate(Routes.proofread(versionId)) },
                onPublished = { versionId ->
                    // 发布成功：详情页替换草稿列表，返回键回到「我的」
                    navController.navigate(Routes.recipeDetail(versionId)) {
                        popUpTo(Routes.MAIN)
                    }
                }
            )
        }
        composable(Routes.DIETARY) {
            DietaryScreen(onBack = { navController.popBackStack() })
        }
        composable(Routes.BILIBILI_AUTH) {
            BilibiliAuthScreen(
                onBack = { navController.popBackStack() },
                onAuthorized = { navController.popBackStack() },
            )
        }
        composable(
            route = Routes.PROOFREAD,
            arguments = listOf(navArgument("versionId") { type = NavType.StringType })
        ) {
            ProofreadScreen(
                onBack = { navController.popBackStack() },
                onPublished = { versionId ->
                    // 发布成功：详情页替换校对页，返回键直接回到首页
                    navController.navigate(Routes.recipeDetail(versionId)) {
                        popUpTo(Routes.MAIN)
                    }
                }
            )
        }
    }
}
