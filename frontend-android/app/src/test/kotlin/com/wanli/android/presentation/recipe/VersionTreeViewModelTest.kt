package com.wanli.android.presentation.recipe

import androidx.lifecycle.SavedStateHandle
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.VersionTreeData
import com.wanli.android.data.remote.VersionTreeNodeDto
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
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class VersionTreeViewModelTest {
    private val dispatcher = StandardTestDispatcher()

    @Before
    fun setUp() = Dispatchers.setMain(dispatcher)

    @After
    fun tearDown() = Dispatchers.resetMain()

    @Test
    fun `load exposes public version nodes`() = runTest {
        val repository = mockk<RecipeRepository>()
        coEvery { repository.getVersionTree("recipe-1") } returns ApiResult.Success(
            VersionTreeData(
                recipe_id = "recipe-1",
                nodes = listOf(
                    VersionTreeNodeDto("source-1", "红烧肉", "SOURCE"),
                    VersionTreeNodeDto("fork-1", "少油红烧肉", "ADAPTED", parent_id = "source-1")
                )
            )
        )

        val viewModel = VersionTreeViewModel(
            SavedStateHandle(mapOf("recipeId" to "recipe-1")),
            repository
        )
        testScheduler.advanceUntilIdle()

        val nodes = (viewModel.uiState.value as VersionTreeUiState.Success).nodes
        assertEquals(2, nodes.size)
        assertEquals("source-1", nodes[1].parent_id)
    }
}
