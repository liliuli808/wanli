package com.wanli.android.presentation.profile

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.BilibiliCredentialData
import com.wanli.android.data.repository.BilibiliCredentialRepository
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.mockk
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.UnconfinedTestDispatcher
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
class BilibiliAuthViewModelTest {

    private val testDispatcher = UnconfinedTestDispatcher()
    private lateinit var repository: BilibiliCredentialRepository

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        repository = mockk()
        coEvery { repository.status() } returns
            ApiResult.Success(BilibiliCredentialData(bound = false))
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `bind success marks authorized`() = runTest {
        coEvery { repository.bind(any()) } returns
            ApiResult.Success(BilibiliCredentialData(bound = true, uname = "小碗", mid = "123"))

        val viewModel = BilibiliAuthViewModel(repository)
        viewModel.bind("sessdata-value-123")

        val state = viewModel.uiState.value
        assertTrue(state.bound)
        assertTrue(state.bindSucceeded)
        assertEquals("小碗", state.uname)
        assertEquals("123", state.mid)
        assertFalse(state.binding)
    }

    @Test
    fun `bind failure surfaces server message`() = runTest {
        coEvery { repository.bind(any()) } returns
            ApiResult.Error("BILIBILI_CREDENTIAL_INVALID", "SESSDATA 无效或已过期，请重新登录 B 站后再试")

        val viewModel = BilibiliAuthViewModel(repository)
        viewModel.bind("expired")

        val state = viewModel.uiState.value
        assertFalse(state.bound)
        assertFalse(state.bindSucceeded)
        assertEquals("SESSDATA 无效或已过期，请重新登录 B 站后再试", state.message)
    }

    @Test
    fun `blank sessdata is ignored`() = runTest {
        val viewModel = BilibiliAuthViewModel(repository)
        viewModel.bind("   ")

        coVerify(exactly = 0) { repository.bind(any()) }
    }

    @Test
    fun `unbind clears bound state`() = runTest {
        coEvery { repository.bind(any()) } returns
            ApiResult.Success(BilibiliCredentialData(bound = true, uname = "小碗", mid = "123"))
        coEvery { repository.unbind() } returns ApiResult.Success(Unit)

        val viewModel = BilibiliAuthViewModel(repository)
        viewModel.bind("sessdata-value-123")
        viewModel.unbind()

        val state = viewModel.uiState.value
        assertFalse(state.bound)
        assertEquals("已解除授权", state.message)
    }
}
