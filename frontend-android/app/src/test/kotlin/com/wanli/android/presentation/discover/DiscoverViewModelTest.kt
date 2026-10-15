package com.wanli.android.presentation.discover

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.RecipeListData
import com.wanli.android.data.remote.RecipeSummaryDto
import com.wanli.android.data.repository.RecipeRepository
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.mockk
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class DiscoverViewModelTest {

    private val testDispatcher = StandardTestDispatcher()
    private lateinit var repository: RecipeRepository

    @Before
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
        repository = mockk()
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun summary(id: String, title: String) = RecipeSummaryDto(
        recipe_id = id,
        title = title,
        recipe_version_id = "version-$id"
    )

    @Test
    fun `init loads latest recipes`() = runTest {
        coEvery { repository.listRecipes("latest", 1, 20) } returns
            ApiResult.Success(RecipeListData(items = listOf(summary("r1", "红烧肉")), total = 1))

        val viewModel = DiscoverViewModel(repository)
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue(state is DiscoverUiState.Success)
        assertEquals("红烧肉", (state as DiscoverUiState.Success).items[0].title)
        assertEquals("latest", state.sort)
    }

    @Test
    fun `setSort hottest reloads list`() = runTest {
        coEvery { repository.listRecipes("latest", 1, 20) } returns
            ApiResult.Success(RecipeListData(items = listOf(summary("r1", "红烧肉"))))
        coEvery { repository.listRecipes("hottest", 1, 20) } returns
            ApiResult.Success(RecipeListData(items = listOf(summary("r2", "宫保鸡丁"))))

        val viewModel = DiscoverViewModel(repository)
        testScheduler.advanceUntilIdle()

        viewModel.setSort("hottest")
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue(state is DiscoverUiState.Success)
        assertEquals("宫保鸡丁", (state as DiscoverUiState.Success).items[0].title)
        assertEquals("hottest", viewModel.currentSort)
    }

    @Test
    fun `setSort same sort does not reload when already success`() = runTest {
        coEvery { repository.listRecipes("latest", 1, 20) } returns
            ApiResult.Success(RecipeListData(items = listOf(summary("r1", "红烧肉"))))

        val viewModel = DiscoverViewModel(repository)
        testScheduler.advanceUntilIdle()

        viewModel.setSort("latest")
        testScheduler.advanceUntilIdle()

        coVerify(exactly = 1) { repository.listRecipes("latest", 1, 20) }
    }

    @Test
    fun `load failure shows error state`() = runTest {
        coEvery { repository.listRecipes(any(), any(), any()) } returns
            ApiResult.Error("NETWORK_ERROR", "网络请求失败")

        val viewModel = DiscoverViewModel(repository)
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue(state is DiscoverUiState.Error)
        assertEquals("网络请求失败", (state as DiscoverUiState.Error).message)
    }
}
