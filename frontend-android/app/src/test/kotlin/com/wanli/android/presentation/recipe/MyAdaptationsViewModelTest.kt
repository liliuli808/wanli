package com.wanli.android.presentation.recipe

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.MyAdaptationDto
import com.wanli.android.data.remote.MyAdaptationsData
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
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class MyAdaptationsViewModelTest {
    private val dispatcher = StandardTestDispatcher()
    private lateinit var repository: RecipeRepository

    @Before
    fun setUp() {
        Dispatchers.setMain(dispatcher)
        repository = mockk()
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun item() = MyAdaptationDto(
        id = "fit-1",
        recipe_id = "recipe-1",
        title = "我的拌面",
        is_public = false
    )

    private fun detail(isPublic: Boolean) = RecipeVersionData(
        id = "fit-1",
        recipe_id = "recipe-1",
        title = "我的拌面",
        version_type = "KITCHEN_FIT",
        is_public = isPublic
    )

    @Test
    fun `load and publish adaptation updates row`() = runTest {
        coEvery { repository.getMyAdaptations() } returns
            ApiResult.Success(MyAdaptationsData(listOf(item())))
        coEvery { repository.setVisibility("fit-1", true) } returns
            ApiResult.Success(detail(true))

        val vm = MyAdaptationsViewModel(repository)
        testScheduler.advanceUntilIdle()
        var state = vm.uiState.value as MyAdaptationsUiState.Success
        assertFalse(state.items.single().is_public)

        vm.setVisibility("fit-1", true)
        testScheduler.advanceUntilIdle()
        state = vm.uiState.value as MyAdaptationsUiState.Success
        assertTrue(state.items.single().is_public)
        assertTrue(state.updatingIds.isEmpty())
        assertNull(state.message)
    }

    @Test
    fun `visibility failure keeps old value and shows error`() = runTest {
        coEvery { repository.getMyAdaptations() } returns
            ApiResult.Success(MyAdaptationsData(listOf(item())))
        coEvery { repository.setVisibility("fit-1", true) } returns
            ApiResult.Error("NETWORK_ERROR", "网络请求失败")

        val vm = MyAdaptationsViewModel(repository)
        testScheduler.advanceUntilIdle()
        vm.setVisibility("fit-1", true)
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value as MyAdaptationsUiState.Success
        assertFalse(state.items.single().is_public)
        assertEquals("网络请求失败", state.message)
    }
}
