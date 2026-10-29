package com.wanli.android.data.repository

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.BaseResponse
import com.wanli.android.data.remote.BilibiliCredentialData
import com.wanli.android.data.remote.BilibiliCredentialResponse
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.mockk
import kotlinx.coroutines.test.runTest
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test
import java.io.IOException

class BilibiliCredentialRepositoryTest {

    private lateinit var apiService: ApiService
    private lateinit var repository: BilibiliCredentialRepository

    @Before
    fun setup() {
        apiService = mockk()
        repository = BilibiliCredentialRepository(apiService)
    }

    @Test
    fun `bind sends sessdata and returns account info`() = runTest {
        coEvery { apiService.bindBilibiliCredential(any()) } returns BilibiliCredentialResponse(
            success = true,
            data = BilibiliCredentialData(bound = true, uname = "小碗", mid = "123"),
        )

        val result = repository.bind("sessdata-value-123")

        assertTrue(result is ApiResult.Success)
        result as ApiResult.Success
        assertEquals("小碗", result.data.uname)
        coVerify { apiService.bindBilibiliCredential(match { it.sessdata == "sessdata-value-123" }) }
    }

    @Test
    fun `status returns bound flag`() = runTest {
        coEvery { apiService.getBilibiliCredential() } returns BilibiliCredentialResponse(
            success = true,
            data = BilibiliCredentialData(bound = true, uname = "小碗", mid = "123"),
        )

        val result = repository.status()

        assertTrue(result is ApiResult.Success)
        assertTrue((result as ApiResult.Success).data.bound)
    }

    @Test
    fun `unbind success returns unit`() = runTest {
        coEvery { apiService.unbindBilibiliCredential() } returns BaseResponse(success = true)

        val result = repository.unbind()

        assertTrue(result is ApiResult.Success)
    }

    @Test
    fun `network failure maps to NETWORK_ERROR`() = runTest {
        coEvery { apiService.getBilibiliCredential() } throws IOException("timeout")

        val result = repository.status()

        assertTrue(result is ApiResult.Error)
        result as ApiResult.Error
        assertEquals("NETWORK_ERROR", result.code)
    }
}
