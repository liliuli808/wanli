package com.wanli.android.presentation.auth

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.UserDto
import com.wanli.android.data.repository.AuthRepository
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
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class LoginViewModelTest {

    private val testDispatcher = UnconfinedTestDispatcher()
    private lateinit var authRepository: AuthRepository

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        authRepository = mockk()
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `login success moves to LoggedIn state`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        viewModel.login("a@b.c", "password123", "cf-token")

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
    }

    @Test
    fun `login network error surfaces error state with message`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Error("NETWORK_ERROR", "网络错误，请稍后重试")

        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        viewModel.login("a@b.c", "password123", "cf-token")

        val state = viewModel.uiState.value
        assertTrue(state is LoginUiState.Error)
        assertEquals("网络错误，请稍后重试", (state as LoginUiState.Error).message)
    }

    @Test
    fun `register success moves to LoggedIn state`() = runTest {
        coEvery { authRepository.register("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        viewModel.register("a@b.c", "password123", "cf-token")

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
    }

    @Test
    fun `captcha url derives from server root`() {
        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        assertEquals("http://10.0.2.2:8081/captcha", viewModel.captchaUrl)
    }

    @Test
    fun `email with control char is sanitized before login`() = runTest {
        coEvery { authRepository.login("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        viewModel.login("a@b.c\u0001", "password123", "cf-token")

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
        coVerify { authRepository.login("a@b.c", "password123", "cf-token") }
    }

    @Test
    fun `register trims email before calling repository`() = runTest {
        coEvery { authRepository.register("a@b.c", "password123", "cf-token") } returns
            ApiResult.Success(UserDto(id = "u1", email = "a@b.c"))

        val viewModel = LoginViewModel(authRepository, "http://10.0.2.2:8081")
        viewModel.register("  a@b.c  ", "password123", "cf-token")

        assertTrue(viewModel.uiState.value is LoginUiState.LoggedIn)
        coVerify { authRepository.register("a@b.c", "password123", "cf-token") }
    }
}
