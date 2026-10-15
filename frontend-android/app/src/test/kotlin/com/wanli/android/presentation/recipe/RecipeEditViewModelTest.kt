package com.wanli.android.presentation.recipe

import androidx.lifecycle.SavedStateHandle
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.CorrectionProposalData
import com.wanli.android.data.remote.IngredientDto
import com.wanli.android.data.remote.RecipeVersionData
import com.wanli.android.data.remote.StepDto
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
class RecipeEditViewModelTest {
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
        id = "version-1",
        recipe_id = "recipe-1",
        title = "红烧肉",
        version_type = "SOURCE",
        is_public = true,
        ingredients = listOf(IngredientDto("五花肉", "500克", "MAIN")),
        steps = listOf(StepDto(1, "切块"))
    )

    @Test
    fun `correction submits full edited copy and reason`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(detail())
        coEvery { repository.submitCorrection("version-1", any()) } returns
            ApiResult.Success(CorrectionProposalData("proposal-1", "PENDING"))

        val viewModel = RecipeEditViewModel(
            SavedStateHandle(mapOf("versionId" to "version-1", "editMode" to EDIT_MODE_CORRECTION)),
            repository
        )
        testScheduler.advanceUntilIdle()
        viewModel.updateNote("视频明确要求焯水")
        viewModel.submit()
        testScheduler.advanceUntilIdle()

        coVerify {
            repository.submitCorrection("version-1", match {
                it.reason == "视频明确要求焯水" &&
                    it.suggested_changes.title == "红烧肉" &&
                    it.suggested_changes.ingredients.single().name == "五花肉"
            })
        }
        assertTrue(viewModel.uiState.value is RecipeEditUiState.Submitted)
    }

    @Test
    fun `submit requires change note`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(detail())
        val viewModel = RecipeEditViewModel(
            SavedStateHandle(mapOf("versionId" to "version-1", "editMode" to EDIT_MODE_FORK)),
            repository
        )
        testScheduler.advanceUntilIdle()
        viewModel.submit()

        val state = viewModel.uiState.value as RecipeEditUiState.Editing
        assertEquals("请完善菜谱并填写改编说明", state.hint)
        coVerify(exactly = 0) { repository.fork(any(), any()) }
    }
}
