package com.wanli.android.presentation.recipe

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.IngredientDto
import com.wanli.android.data.remote.MyRecipeDraftsData
import com.wanli.android.data.remote.PublishData
import com.wanli.android.data.remote.RecipeDraftDto
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
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class MyDraftsViewModelTest {
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

    private fun draft(versionId: String = "ver-1") = RecipeDraftDto(
        recipe_id = "recipe-1",
        recipe_version_id = versionId,
        title = "番茄炒蛋",
        bvid = "BV1gz411B7yY"
    )

    private fun detail(versionId: String = "ver-1") = RecipeVersionData(
        id = versionId,
        recipe_id = "recipe-1",
        title = "番茄炒蛋",
        version_type = "SOURCE",
        is_public = false,
        servings = "2人份",
        ingredients = listOf(
            IngredientDto(name = "鸡蛋", amount = "2个", category = "MAIN", is_explicit = true),
            IngredientDto(name = "盐", amount = null, category = "SEASONING", is_explicit = true)
        ),
        steps = listOf(
            StepDto(step_order = 1, description = "鸡蛋打散", video_timestamp = 24, heat = null)
        )
    )

    @Test
    fun `load exposes drafts`() = runTest {
        coEvery { repository.getMyRecipeDrafts() } returns
            ApiResult.Success(MyRecipeDraftsData(listOf(draft())))

        val vm = MyDraftsViewModel(repository)
        dispatcher.scheduler.advanceUntilIdle()

        val state = vm.uiState.value as MyDraftsUiState.Success
        assertEquals(1, state.items.size)
        assertEquals("BV1gz411B7yY", state.items[0].bvid)
    }

    @Test
    fun `publish submits detail content and removes draft`() = runTest {
        coEvery { repository.getMyRecipeDrafts() } returns
            ApiResult.Success(MyRecipeDraftsData(listOf(draft(), draft("ver-2"))))
        coEvery { repository.getVersionDetail("ver-1") } returns ApiResult.Success(detail())
        coEvery { repository.publish("ver-1", any()) } returns
            ApiResult.Success(PublishData(recipe_version_id = "ver-1", is_public = true))

        val vm = MyDraftsViewModel(repository)
        dispatcher.scheduler.advanceUntilIdle()
        vm.publish(draft())
        dispatcher.scheduler.advanceUntilIdle()

        // 发布请求必须带上草稿的完整内容，而不是列表里的空表单
        coVerify {
            repository.publish("ver-1", match { request ->
                request.title == "番茄炒蛋" &&
                    request.ingredients.size == 2 &&
                    request.steps.size == 1 &&
                    request.steps[0].video_timestamp == 24
            })
        }

        val state = vm.uiState.value as MyDraftsUiState.Success
        assertEquals(listOf("ver-2"), state.items.map { it.recipe_version_id })
        assertTrue(state.publishingIds.isEmpty())
        assertEquals("ver-1", state.publishedVersionId)
        assertFalse(state.messageIsError)
    }

    @Test
    fun `publish failure keeps draft and reports error`() = runTest {
        coEvery { repository.getMyRecipeDrafts() } returns
            ApiResult.Success(MyRecipeDraftsData(listOf(draft())))
        coEvery { repository.getVersionDetail("ver-1") } returns ApiResult.Success(detail())
        coEvery { repository.publish("ver-1", any()) } returns
            ApiResult.Error(code = "AI_PARSE_FAILED", message = "发布失败，请稍后再试")

        val vm = MyDraftsViewModel(repository)
        dispatcher.scheduler.advanceUntilIdle()
        vm.publish(draft())
        dispatcher.scheduler.advanceUntilIdle()

        val state = vm.uiState.value as MyDraftsUiState.Success
        assertEquals(1, state.items.size)
        assertEquals("发布失败，请稍后再试", state.message)
        assertTrue(state.messageIsError)
        assertNull(state.publishedVersionId)
    }

    @Test
    fun `incomplete draft is not published`() = runTest {
        coEvery { repository.getMyRecipeDrafts() } returns
            ApiResult.Success(MyRecipeDraftsData(listOf(draft())))
        coEvery { repository.getVersionDetail("ver-1") } returns
            ApiResult.Success(detail().copy(ingredients = emptyList()))

        val vm = MyDraftsViewModel(repository)
        dispatcher.scheduler.advanceUntilIdle()
        vm.publish(draft())
        dispatcher.scheduler.advanceUntilIdle()

        coVerify(exactly = 0) { repository.publish(any(), any()) }
        val state = vm.uiState.value as MyDraftsUiState.Success
        assertEquals(1, state.items.size)
        assertTrue(state.messageIsError)
    }

    @Test
    fun `consumePublished clears navigation trigger`() = runTest {
        coEvery { repository.getMyRecipeDrafts() } returns
            ApiResult.Success(MyRecipeDraftsData(listOf(draft())))
        coEvery { repository.getVersionDetail("ver-1") } returns ApiResult.Success(detail())
        coEvery { repository.publish("ver-1", any()) } returns
            ApiResult.Success(PublishData(recipe_version_id = "ver-1", is_public = true))

        val vm = MyDraftsViewModel(repository)
        dispatcher.scheduler.advanceUntilIdle()
        vm.publish(draft())
        dispatcher.scheduler.advanceUntilIdle()
        vm.consumePublished()

        assertNull((vm.uiState.value as MyDraftsUiState.Success).publishedVersionId)
    }
}
