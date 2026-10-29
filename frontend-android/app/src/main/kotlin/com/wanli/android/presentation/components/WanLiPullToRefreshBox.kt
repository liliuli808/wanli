package com.wanli.android.presentation.components

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.BoxScope
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.pulltorefresh.PullToRefreshContainer
import androidx.compose.material3.pulltorefresh.rememberPullToRefreshState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clipToBounds
import androidx.compose.ui.input.nestedscroll.nestedScroll

/**
 * 统一下拉刷新容器：包住可滚动内容，下拉触发 [onRefresh]。
 *
 * [isRefreshing] 传 ViewModel 的加载态（如 uiState is Loading），
 * 加载结束时刷新指示器自动收起。
 */
@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun WanLiPullToRefreshBox(
    isRefreshing: Boolean,
    onRefresh: () -> Unit,
    modifier: Modifier = Modifier,
    content: @Composable BoxScope.() -> Unit
) {
    val state = rememberPullToRefreshState()

    LaunchedEffect(isRefreshing) {
        if (isRefreshing) state.startRefresh() else state.endRefresh()
    }
    LaunchedEffect(state.isRefreshing) {
        if (state.isRefreshing) onRefresh()
    }

    // clipToBounds：指示器闲置时被上移到容器外，不裁剪会在顶栏下方留一个白圆点
    Box(modifier.nestedScroll(state.nestedScrollConnection).clipToBounds()) {
        content()
        PullToRefreshContainer(
            state = state,
            modifier = Modifier.align(Alignment.TopCenter)
        )
    }
}

/**
 * 让空态/错误态也能触发下拉刷新：撑满并可垂直滚动（内容少时不显示滚动条）。
 * 下拉刷新的嵌套滚动依赖可滚动子项，静态页面需用它包一层。
 */
@Composable
fun ScrollableFill(content: @Composable () -> Unit) {
    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState()),
        verticalArrangement = Arrangement.Center,
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        content()
    }
}
