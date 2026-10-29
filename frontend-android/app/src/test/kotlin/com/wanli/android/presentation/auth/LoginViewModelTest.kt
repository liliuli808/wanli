package com.wanli.android.presentation.auth

import com.wanli.android.data.local.TokenDataStore
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.UserDto
import com.wanli.android.data.repository.AuthRepository
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.just
import io.mockk.mockk
import io.mockk.Runs
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

@OptIn(ExperimentalCoroutinesApi::class)
class LoginViewModelTest {

    private val testDispatcher = UnconfinedTestDispatcher()
    private lateinit var authRepository: AuthRepository
    private lateinit var tokenDataStore: TokenDataStore

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        authRepository = mockk()
        tokenDataStore = mockk(relaxed = true)
    }

    private fun viewModel() = LoginViewModel(authRepository, tokenDataStore, "http://10.0.2.2:8081")

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `login success moves to LoggedIn state`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = viewModel()
        viewModel.login("a@b.c", "password123", "cf-token", remember = true)

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
    }

    @Test
    fun `login network error surfaces error state with message`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Error("NETWORK_ERROR", "网络错误，请稍后重试")

        val viewModel = viewModel()
        viewModel.login("a@b.c", "password123", "cf-token", remember = true)

        val state = viewModel.uiState.value
        assertTrue(state is LoginUiState.Error)
        assertEquals("网络错误，请稍后重试", (state as LoginUiState.Error).message)
    }

    @Test
    fun `register success moves to LoggedIn state`() = runTest {
        coEvery { authRepository.register("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = viewModel()
        viewModel.register("a@b.c", "password123", "cf-token", remember = true)

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
    }

    @Test
    fun `captcha url derives from server root`() {
        val viewModel = viewModel()
        assertEquals("http://10.0.2.2:8081/captcha", viewModel().captchaUrl)
    }

    @Test
    fun `email with control char is sanitized before login`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = viewModel()
        viewModel.login("a@b.c\u0001", "password123", "cf-token", remember = true)

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
        coVerify { authRepository.login("a@b.c", "password123", "cf-token") }
    }

    @Test
    fun `register trims email before calling repository`() = runTest {
        coEvery { authRepository.register("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = viewModel()
        viewModel.register("  a@b.c  ", "password123", "cf-token", remember = true)

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
        coVerify { authRepository.register("a@b.c", "password123", "cf-token") }
    }

    @Test
    fun `login with remember persists sanitized credentials`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))
        coEvery { tokenDataStore.saveCredentials(any(), any()) } just Runs

        viewModel().login(" a@b.c\u0001 ", "password123", "cf-token", remember = true)

        coVerify { tokenDataStore.saveCredentials("a@b.c", "password123") }
    }

    @Test
    fun `login without remember clears saved credentials`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))
        coEvery { tokenDataStore.clearCredentials() } just Runs

        viewModel().login("a@b.c", "password123", "cf-token", remember = false)

        coVerify { tokenDataStore.clearCredentials() }
        coVerify(exactly = 0) { tokenDataStore.saveCredentials(any(), any()) }
    }

    @Test
    fun `failed login does not touch saved credentials`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Error("INVALID_CREDENTIALS", "邮箱或密码错误")

        viewModel().login("a@b.c", "password123", "cf-token", remember = true)

        coVerify(exactly = 0) { tokenDataStore.saveCredentials(any(), any()) }
        coVerify(exactly = 0) { tokenDataStore.clearCredentials() }
    }
}
