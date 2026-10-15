package com.wanli.android.presentation.recipe

import androidx.lifecycle.SavedStateHandle
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.RecipeVersionData
import com.wanli.android.data.repository.RecipeRepository
import io.mockk.coEvery
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
class RecipeDetailViewModelTest {

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

    private fun viewModel(versionId: String = "version-1") =
        RecipeDetailViewModel(SavedStateHandle(mapOf("versionId" to versionId)), repository)

    private fun detail(videoStatus: String, warning: String? = null) = RecipeVersionData(
        id = "version-1",
        recipe_id = "recipe-1",
        title = "红烧肉",
        video_status = videoStatus,
        video_warning = warning,
        version_type = "SOURCE",
        is_public = true
    )

    @Test
    fun `load success shows detail`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns
            ApiResult.Success(detail("ACTIVE"))

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value
        assertTrue(state is RecipeDetailUiState.Success)
        assertEquals("红烧肉", (state as RecipeDetailUiState.Success).detail.title)
    }

    @Test
    fun `expired video detail carries warning (AC4)`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns
            ApiResult.Success(detail("EXPIRED", "原视频已下架或失效，菜谱内容仍可正常浏览"))

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value
        assertTrue(state is RecipeDetailUiState.Success)
        val data = (state as RecipeDetailUiState.Success).detail
        assertEquals("EXPIRED", data.video_status)
        assertEquals("原视频已下架或失效，菜谱内容仍可正常浏览", data.video_warning)
    }

    @Test
    fun `load failure shows error and retry works`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returnsMany listOf(
            ApiResult.Error("RESOURCE_NOT_FOUND", "资源不存在"),
            ApiResult.Success(detail("ACTIVE"))
        )

        val vm = viewModel()
        testScheduler.advanceUntilIdle()
        assertTrue(vm.uiState.value is RecipeDetailUiState.Error)

        vm.load()
        testScheduler.advanceUntilIdle()
        assertTrue(vm.uiState.value is RecipeDetailUiState.Success)
    }

    @Test
    fun `formatTimestamp renders mm ss`() {
        assertEquals("0:45", formatTimestamp(45))
        assertEquals("2:05", formatTimestamp(125))
        assertEquals("10:00", formatTimestamp(600))
    }
}
