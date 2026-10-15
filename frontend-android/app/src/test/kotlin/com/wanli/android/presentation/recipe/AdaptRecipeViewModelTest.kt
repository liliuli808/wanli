package com.wanli.android.presentation.recipe

import androidx.lifecycle.SavedStateHandle
import com.wanli.android.data.remote.AdaptationNotesDto
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.RecipeVersionData
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
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class AdaptRecipeViewModelTest {
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

    private fun detail() = RecipeVersionData(
        id = "fit-1",
        recipe_id = "recipe-1",
        parent_id = "source-1",
        title = "芝麻酱拌面",
        version_type = "KITCHEN_FIT",
        is_public = false,
        adaptation_notes = AdaptationNotesDto(flavor_changes = listOf("香气会有变化"))
    )

    private fun viewModel() = AdaptRecipeViewModel(
        SavedStateHandle(mapOf("versionId" to "source-1")),
        repository
    )

    @Test
    fun `adapt saves private kitchen fit and exposes notes`() = runTest {
        coEvery { repository.adapt("source-1") } returns ApiResult.Success(detail())
        val vm = viewModel()

        assertTrue(vm.uiState.value is AdaptRecipeUiState.Ready)
        vm.adapt()
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value as AdaptRecipeUiState.Success
        assertFalse(state.detail.is_public)
        assertEquals("香气会有变化", state.detail.adaptation_notes?.flavor_changes?.single())
        coVerify(exactly = 1) { repository.adapt("source-1") }
    }

    @Test
    fun `allergy block uses explicit safety warning`() = runTest {
        coEvery { repository.adapt("source-1") } returns ApiResult.Error(
            code = "ALLERGY_BLOCKED",
            message = "由于安全限制，该菜谱无法适配"
        )
        val vm = viewModel()

        vm.adapt()
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value as AdaptRecipeUiState.Error
        assertEquals("ALLERGY_BLOCKED", state.code)
        assertTrue(state.message.contains("建议放弃制作"))
    }
}
