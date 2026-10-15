package com.wanli.android.data.repository

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class HealthRepository @Inject constructor(
    private val apiService: ApiService
) {
    suspend fun checkHealth(): ApiResult<String> {
        return try {
            val response = apiService.health()
            if (response.success && response.data != null) {
                ApiResult.Success(response.data.status)
            } else {
                ApiResult.Error(
                    code = response.error?.code ?: "UNKNOWN_ERROR",
                    message = response.error?.message ?: "未知错误"
                )
            }
        } catch (e: Exception) {
            ApiResult.Error(
                code = "NETWORK_ERROR",
                message = e.message ?: "网络请求失败"
            )
        }
    }
}
