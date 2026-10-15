package com.wanli.android.presentation.recipe

import androidx.lifecycle.SavedStateHandle
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.IngredientDto
import com.wanli.android.data.remote.PublishData
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
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class ProofreadViewModelTest {

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

    private fun draftDetail() = RecipeVersionData(
        id = "version-1",
        recipe_id = "recipe-1",
        title = "红烧肉",
        version_type = "SOURCE",
        is_public = false,
        servings = "2人份",
        estimated_minutes = 45,
        ingredients = listOf(
            IngredientDto(name = "五花肉", amount = "500克", category = "MAIN", is_explicit = true)
        ),
        steps = listOf(
            StepDto(step_order = 1, description = "切块", video_timestamp = 45)
        )
    )

    private fun viewModel() =
        ProofreadViewModel(SavedStateHandle(mapOf("versionId" to "version-1")), repository)

    @Test
    fun `load populates form from draft`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(draftDetail())

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value
        assertTrue(state is ProofreadUiState.Editing)
        val form = (state as ProofreadUiState.Editing).form
        assertEquals("红烧肉", form.title)
        assertEquals("2人份", form.servings)
        assertEquals("45", form.estimatedMinutes)
        assertEquals("五花肉", form.ingredients[0].name)
        assertEquals("500克", form.ingredients[0].amount)
        assertEquals("切块", form.steps[0].description)
        assertEquals("45", form.steps[0].videoTimestamp)
    }

    @Test
    fun `ingredient and step editing updates form`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(draftDetail())

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.addIngredient()
        vm.updateIngredient(1, IngredientForm(name = "冰糖", amount = "20克", category = CATEGORY_SEASONING))
        vm.addStep()
        vm.updateStep(1, StepForm(description = "炒糖色", videoTimestamp = "90"))

        val form = (vm.uiState.value as ProofreadUiState.Editing).form
        assertEquals(2, form.ingredients.size)
        assertEquals("冰糖", form.ingredients[1].name)
        assertEquals(CATEGORY_SEASONING, form.ingredients[1].category)
        assertEquals(2, form.steps.size)
        assertEquals("炒糖色", form.steps[1].description)

        vm.removeStep(0)
        val afterRemove = (vm.uiState.value as ProofreadUiState.Editing).form
        assertEquals(1, afterRemove.steps.size)
        assertEquals("炒糖色", afterRemove.steps[0].description)
    }

    @Test
    fun `save with blank title shows hint without network call`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(draftDetail())

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.updateForm { it.copy(title = "  ") }
        vm.save()
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value as ProofreadUiState.Editing
        assertNotNull(state.hint)
        assertTrue(state.isError)
        coVerify(exactly = 0) { repository.updateDraft(any(), any()) }
    }

    @Test
    fun `save valid form calls updateDraft with assembled request`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(draftDetail())
        coEvery { repository.updateDraft("version-1", any()) } returns ApiResult.Success(draftDetail())

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.updateForm { it.copy(title = "秘制红烧肉") }
        vm.save()
        testScheduler.advanceUntilIdle()

        coVerify(exactly = 1) {
            repository.updateDraft("version-1", match {
                it.title == "秘制红烧肉" &&
                    it.ingredients.size == 1 &&
                    it.ingredients[0].name == "五花肉" &&
                    it.steps.size == 1 &&
                    it.steps[0].step_order == 1 &&
                    it.steps[0].video_timestamp == 45
            })
        }
        val state = vm.uiState.value as ProofreadUiState.Editing
        assertEquals("草稿已保存", state.hint)
        assertFalse(state.isError)
    }

    @Test
    fun `save success preserves edits made while request in flight`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(draftDetail())
        coEvery { repository.updateDraft("version-1", any()) } returns ApiResult.Success(draftDetail())

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.save()
        // 请求尚未返回时继续输入（StandardTestDispatcher 下协程未调度）
        vm.updateForm { it.copy(title = "秘制红烧肉") }
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value as ProofreadUiState.Editing
        assertEquals("秘制红烧肉", state.form.title)
        assertEquals("草稿已保存", state.hint)
        assertFalse(state.isError)
        assertFalse(state.working)
    }

    @Test
    fun `save failure keeps current form and marks error`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(draftDetail())
        coEvery { repository.updateDraft("version-1", any()) } returns ApiResult.Error("NETWORK", "网络异常")

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.save()
        vm.updateForm { it.copy(title = "改过的菜名") }
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value as ProofreadUiState.Editing
        assertEquals("改过的菜名", state.form.title)
        assertEquals("网络异常", state.hint)
        assertTrue(state.isError)
        assertFalse(state.working)
    }

    @Test
    fun `consumePublished returns to editing with preserved form`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(draftDetail())
        coEvery { repository.publish("version-1", any()) } returns
            ApiResult.Success(PublishData(recipe_version_id = "version-1", is_public = true))

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.updateForm { it.copy(title = "秘制红烧肉") }
        vm.publish()
        testScheduler.advanceUntilIdle()
        assertTrue(vm.uiState.value is ProofreadUiState.Published)

        vm.consumePublished()

        val state = vm.uiState.value as ProofreadUiState.Editing
        assertEquals("秘制红烧肉", state.form.title)
        assertFalse(state.working)
    }

    @Test
    fun `publish success enters published state`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(draftDetail())
        coEvery { repository.publish("version-1", any()) } returns
            ApiResult.Success(PublishData(recipe_version_id = "version-1", is_public = true))

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.publish()
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value
        assertTrue(state is ProofreadUiState.Published)
        assertEquals("version-1", (state as ProofreadUiState.Published).versionId)
    }

    @Test
    fun `publish failure stays editing with hint`() = runTest {
        coEvery { repository.getVersionDetail("version-1") } returns ApiResult.Success(draftDetail())
        coEvery { repository.publish("version-1", any()) } returns
            ApiResult.Error("FORBIDDEN", "仅版本作者可以执行此操作")

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.publish()
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value
        assertTrue(state is ProofreadUiState.Editing)
        assertEquals("仅版本作者可以执行此操作", (state as ProofreadUiState.Editing).hint)
    }

    @Test
    fun `buildRequest validation rules`() {
        val valid = ProofreadForm(
            title = "红烧肉",
            ingredients = listOf(IngredientForm(name = "五花肉")),
            steps = listOf(StepForm(description = "切块"))
        )
        assertNotNull(ProofreadViewModel.buildRequest(valid))

        assertNull(ProofreadViewModel.buildRequest(valid.copy(title = "")))
        assertNull(ProofreadViewModel.buildRequest(valid.copy(ingredients = listOf(IngredientForm(name = " ")))))
        assertNull(ProofreadViewModel.buildRequest(valid.copy(steps = emptyList())))

        // 空白行被过滤，步骤序号重排
        val request = ProofreadViewModel.buildRequest(
            valid.copy(
                ingredients = listOf(IngredientForm(name = ""), IngredientForm(name = "五花肉")),
                steps = listOf(StepForm(description = " "), StepForm(description = "切块"))
            )
        )
        assertNotNull(request)
        assertEquals(1, request!!.ingredients.size)
        assertEquals(1, request.steps.size)
        assertEquals(1, request.steps[0].step_order)
    }
}
