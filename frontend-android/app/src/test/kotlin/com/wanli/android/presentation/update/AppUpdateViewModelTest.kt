package com.wanli.android.presentation.update

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.repository.AppUpdateRepository
import com.wanli.android.domain.model.AppUpdateInfo
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import io.mockk.Runs
import io.mockk.verify
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.File

@OptIn(ExperimentalCoroutinesApi::class)
class AppUpdateViewModelTest {

    private val testDispatcher = UnconfinedTestDispatcher()
    private lateinit var repository: AppUpdateRepository

    private val newerInfo = AppUpdateInfo(
        versionCode = 99,
        versionName = "0.0.99",
        size = 1024,
        sha256 = "abc",
        commit = "d830e28",
        downloadUrl = "https://api.example.com/api/v1/app/download",
    )

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        repository = mockk()
        every { repository.wasDismissedRecently(any()) } returns false
        every { repository.markDismissed(any()) } just Runs
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `launch check shows dialog when newer version exists`() = runTest {
        coEvery { repository.check() } returns ApiResult.Success(newerInfo)

        val viewModel = AppUpdateViewModel(repository)
        viewModel.checkOnLaunch()

        assertEquals(AppUpdateUiState.Available(newerInfo), viewModel.uiState.value)
    }

    @Test
    fun `launch check stays idle when version was dismissed recently`() = runTest {
        coEvery { repository.check() } returns ApiResult.Success(newerInfo)
        every { repository.wasDismissedRecently(99) } returns true

        val viewModel = AppUpdateViewModel(repository)
        viewModel.checkOnLaunch()

        assertEquals(AppUpdateUiState.Idle, viewModel.uiState.value)
    }

    @Test
    fun `launch check ignores network errors`() = runTest {
        coEvery { repository.check() } returns ApiResult.Error("NETWORK_ERROR", "网络不可用")

        val viewModel = AppUpdateViewModel(repository)
        viewModel.checkOnLaunch()

        assertEquals(AppUpdateUiState.Idle, viewModel.uiState.value)
    }

    @Test
    fun `launch check runs only once`() = runTest {
        coEvery { repository.check() } returns ApiResult.Success(null)

        val viewModel = AppUpdateViewModel(repository)
        viewModel.checkOnLaunch()
        viewModel.checkOnLaunch()

        coVerify(exactly = 1) { repository.check() }
    }

    @Test
    fun `manual check reports up to date`() = runTest {
        coEvery { repository.check() } returns ApiResult.Success(null)

        val viewModel = AppUpdateViewModel(repository)
        viewModel.check(manual = true)

        assertTrue(viewModel.uiState.value is AppUpdateUiState.UpToDate)
    }

    @Test
    fun `manual check failure is shown to user`() = runTest {
        coEvery { repository.check() } returns ApiResult.Error("APP_UPDATE_UNAVAILABLE", "暂时无法获取最新版本")

        val viewModel = AppUpdateViewModel(repository)
        viewModel.check(manual = true)

        val state = viewModel.uiState.value as AppUpdateUiState.Failed
        assertEquals("暂时无法获取最新版本", state.message)
    }

    @Test
    fun `manual check with autoDownload starts download immediately`() = runTest {
        coEvery { repository.check() } returns ApiResult.Success(newerInfo)
        val apk = File.createTempFile("wanli-99-auto", ".apk").apply { deleteOnExit() }
        coEvery { repository.download(newerInfo, any()) } returns ApiResult.Success(apk)

        val viewModel = AppUpdateViewModel(repository)
        viewModel.check(manual = true, autoDownload = true)

        // 不需要用户再点一次「立即更新」，直接进入 Ready
        assertEquals(AppUpdateUiState.Ready(newerInfo, apk), viewModel.uiState.value)
        coVerify(exactly = 1) { repository.download(newerInfo, any()) }
    }

    @Test
    fun `manual check without autoDownload waits for user`() = runTest {
        coEvery { repository.check() } returns ApiResult.Success(newerInfo)

        val viewModel = AppUpdateViewModel(repository)
        viewModel.check(manual = true)

        assertTrue(viewModel.uiState.value is AppUpdateUiState.Available)
        coVerify(exactly = 0) { repository.download(any(), any()) }
    }

    @Test
    fun `dismiss records version and hides dialog`() = runTest {
        coEvery { repository.check() } returns ApiResult.Success(newerInfo)
        val viewModel = AppUpdateViewModel(repository)
        viewModel.checkOnLaunch()

        viewModel.dismiss()

        verify { repository.markDismissed(99) }
        assertEquals(AppUpdateUiState.Idle, viewModel.uiState.value)
    }

    @Test
    fun `download success yields ready state`() = runTest {
        coEvery { repository.check() } returns ApiResult.Success(newerInfo)
        val apk = File.createTempFile("wanli-99", ".apk").apply { deleteOnExit() }
        coEvery { repository.download(any(), any()) } returns ApiResult.Success(apk)

        val viewModel = AppUpdateViewModel(repository)
        viewModel.checkOnLaunch()
        viewModel.download()

        assertEquals(AppUpdateUiState.Ready(newerInfo, apk), viewModel.uiState.value)
    }

    @Test
    fun `download failure keeps info for retry`() = runTest {
        coEvery { repository.check() } returns ApiResult.Success(newerInfo)
        coEvery { repository.download(any(), any()) } returns
            ApiResult.Error("CHECKSUM_MISMATCH", "安装包校验失败，请重试")

        val viewModel = AppUpdateViewModel(repository)
        viewModel.checkOnLaunch()
        viewModel.download()

        val state = viewModel.uiState.value as AppUpdateUiState.Failed
        assertEquals(newerInfo, state.info)
        assertEquals("安装包校验失败，请重试", state.message)
    }
}
