package com.wanli.android.data.repository

import com.wanli.android.data.local.TokenDataStore
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.LoginData
import com.wanli.android.data.remote.LoginResponse
import com.wanli.android.data.remote.RegisterRequest
import com.wanli.android.data.remote.UserDto
import io.mockk.Runs
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.every
import io.mockk.just
import io.mockk.mockk
import kotlinx.coroutines.flow.flowOf
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.IOException

class AuthRepositoryTest {

    private lateinit var apiService: ApiService
    private lateinit var tokenDataStore: TokenDataStore
    private lateinit var repository: AuthRepository

    @Before
    fun setup() {
        apiService = mockk()
        tokenDataStore = mockk()
        every { tokenDataStore.isLoggedIn } returns flowOf(false)
        coEvery { tokenDataStore.saveToken(any(), any(), any()) } just Runs
        repository = AuthRepository(apiService, tokenDataStore)
    }

    @Test
    fun `login returns network error instead of throwing when api call fails`() = runTest {
        coEvery { apiService.login(any()) } throws IOException("timeout")

        val result = repository.login("a@b.c", "password123", "cf-token")

        assertTrue(result is ApiResult.Error)
        result as ApiResult.Error
        assertEquals("NETWORK_ERROR", result.code)
        assertEquals("timeout", result.message)
    }

    @Test
    fun `register sends email password and turnstile token`() = runTest {
        coEvery { apiService.register(any()) } returns mockLoginResponse()

        val result = repository.register("a@b.c", "password123", "cf-token")

        assertTrue(result is ApiResult.Success)
        result as ApiResult.Success
        assertEquals("u1", result.data.id)
        coVerify { tokenDataStore.saveToken("jwt-token", "u1", "a@b.c") }
        coVerify { apiService.register(match<RegisterRequest> {
            it.email == "a@b.c" && it.password == "password123" && it.turnstileToken == "cf-token"
        }) }
    }
}

private fun mockLoginResponse() = LoginResponse(
    success = true,
    data = LoginData(
        token = "jwt-token",
        user = UserDto(id = "u1", email = "a@b.c")
    )
)
