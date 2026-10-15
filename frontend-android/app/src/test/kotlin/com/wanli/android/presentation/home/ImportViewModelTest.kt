package com.wanli.android.presentation.home

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.VideoImportData
import com.wanli.android.data.remote.VideoImportTaskData
import com.wanli.android.data.repository.VideoImportRepository
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
class ImportViewModelTest {

    private val testDispatcher = StandardTestDispatcher()
    private lateinit var repository: VideoImportRepository
    private lateinit var viewModel: ImportViewModel

    @Before
    fun setUp() {
        Dispatchers.setMain(testDispatcher)
        repository = mockk()
        viewModel = ImportViewModel(repository)
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun taskData(
        status: String,
        message: String? = null,
        errorCode: String? = null,
        versionId: String? = null
    ) = VideoImportTaskData(
        task_id = "task-1",
        status = status,
        bvid = "BV1xx411c7xx",
        message = message,
        error_code = errorCode,
        recipe_version_id = versionId
    )

    @Test
    fun `submit blank url fails immediately without network call`() = runTest {
        viewModel.submit("   ")

        val state = viewModel.uiState.value
        assertTrue(state is ImportUiState.Failed)
        assertEquals("请先粘贴 B 站视频链接", (state as ImportUiState.Failed).message)
    }

    @Test
    fun `submit success polls until completed`() = runTest {
        coEvery { repository.submit(any()) } returns
            VideoImportRepository.SubmitResult.Success(VideoImportData("task-1", "PENDING", "BV1xx411c7xx"))
        coEvery { repository.getTask("task-1") } returnsMany listOf(
            ApiResult.Success(taskData("FETCHING", "正在获取视频信息")),
            ApiResult.Success(taskData("PARSING", "正在用AI拆解菜谱")),
            ApiResult.Success(taskData("COMPLETED", versionId = "version-1"))
        )

        viewModel.submit("https://www.bilibili.com/video/BV1xx411c7xx/")
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue(state is ImportUiState.Completed)
        assertEquals("version-1", (state as ImportUiState.Completed).recipeVersionId)
    }

    @Test
    fun `submit shows failed state when task fails`() = runTest {
        coEvery { repository.submit(any()) } returns
            VideoImportRepository.SubmitResult.Success(VideoImportData("task-1", "PENDING", "BV1xx411c7xx"))
        coEvery { repository.getTask("task-1") } returns
            ApiResult.Success(taskData("FAILED", message = "该视频无可用文本来源", errorCode = "SUBTITLE_NOT_AVAILABLE"))

        viewModel.submit("BV1xx411c7xx")
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue(state is ImportUiState.Failed)
        assertEquals("该视频无可用文本来源", (state as ImportUiState.Failed).message)
        assertEquals("SUBTITLE_NOT_AVAILABLE", state.errorCode)
    }

    @Test
    fun `duplicate submission shows duplicate state with recipe id`() = runTest {
        coEvery { repository.submit(any()) } returns
            VideoImportRepository.SubmitResult.Duplicate(
                recipeId = "recipe-1",
                recipeVersionId = "version-1",
                message = "该视频已有公开来源解析版"
            )

        viewModel.submit("BV1xx411c7xx")
        testScheduler.advanceUntilIdle()

        val state = viewModel.uiState.value
        assertTrue(state is ImportUiState.Duplicate)
        assertEquals("recipe-1", (state as ImportUiState.Duplicate).recipeId)
        assertEquals("version-1", state.recipeVersionId)
    }

    @Test
    fun `unauthorized submission requires login`() = runTest {
        coEvery { repository.submit(any()) } returns
            VideoImportRepository.SubmitResult.Error(code = "UNAUTHORIZED", message = "未登录")

        viewModel.submit("BV1xx411c7xx")
        testScheduler.advanceUntilIdle()

        assertTrue(viewModel.uiState.value is ImportUiState.LoginRequired)
    }

    @Test
    fun `transient network error during polling does not break polling`() = runTest {
        coEvery { repository.submit(any()) } returns
            VideoImportRepository.SubmitResult.Success(VideoImportData("task-1", "PENDING", "BV1xx411c7xx"))
        coEvery { repository.getTask("task-1") } returnsMany listOf(
            ApiResult.Error("NETWORK_ERROR", "timeout"),
            ApiResult.Success(taskData("COMPLETED", versionId = "version-1"))
        )

        viewModel.submit("BV1xx411c7xx")
        testScheduler.advanceUntilIdle()

        assertTrue(viewModel.uiState.value is ImportUiState.Completed)
    }

    @Test
    fun `reset returns to idle`() = runTest {
        viewModel.submit("")
        assertTrue(viewModel.uiState.value is ImportUiState.Failed)

        viewModel.reset()
        assertTrue(viewModel.uiState.value is ImportUiState.Idle)
    }

    @Test
    fun `failureMessage falls back by error code`() {
        assertEquals(
            "该视频没有可用字幕，无法解析",
            ImportViewModel.failureMessage(null, "SUBTITLE_NOT_AVAILABLE")
        )
        assertEquals("自定义文案", ImportViewModel.failureMessage("自定义文案", "AI_PARSE_FAILED"))
        assertEquals("解析失败，请稍后再试", ImportViewModel.failureMessage("", "UNKNOWN"))
    }
}
