package com.wanli.android.presentation.recipe

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.PendingCorrectionDto
import com.wanli.android.data.remote.PendingCorrectionsData
import com.wanli.android.data.remote.ReviewCorrectionData
import com.wanli.android.data.remote.UpdateDraftRequest
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
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class PendingCorrectionsViewModelTest {
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

    private fun proposal() = PendingCorrectionDto(
        id = "proposal-1",
        recipe_version_id = "version-1",
        recipe_title = "红烧肉",
        reason = "漏了焯水",
        suggested_changes = UpdateDraftRequest("红烧肉", ingredients = emptyList(), steps = emptyList())
    )

    @Test
    fun `approve removes reviewed proposal`() = runTest {
        coEvery { repository.getPendingCorrections() } returns
            ApiResult.Success(PendingCorrectionsData(listOf(proposal())))
        coEvery { repository.reviewCorrection("proposal-1", any()) } returns
            ApiResult.Success(ReviewCorrectionData("approve", "version-2"))

        val viewModel = PendingCorrectionsViewModel(repository)
        testScheduler.advanceUntilIdle()
        viewModel.review("proposal-1", approve = true)
        testScheduler.advanceUntilIdle()

        assertEquals(0, (viewModel.uiState.value as PendingCorrectionsUiState.Success).items.size)
        coVerify { repository.reviewCorrection("proposal-1", match { it.action == "approve" }) }
    }

    @Test
    fun `reject requires reason`() = runTest {
        coEvery { repository.getPendingCorrections() } returns
            ApiResult.Success(PendingCorrectionsData(listOf(proposal())))
        val viewModel = PendingCorrectionsViewModel(repository)
        testScheduler.advanceUntilIdle()
        viewModel.review("proposal-1", approve = false)

        assertEquals(
            "拒绝时请填写理由",
            (viewModel.uiState.value as PendingCorrectionsUiState.Success).message
        )
        coVerify(exactly = 0) { repository.reviewCorrection(any(), any()) }
    }
}
