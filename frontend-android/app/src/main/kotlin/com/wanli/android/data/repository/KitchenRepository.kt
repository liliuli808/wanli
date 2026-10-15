package com.wanli.android.data.repository

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.KitchenData
import com.wanli.android.data.remote.RestrictionDto
import com.wanli.android.data.remote.UpdateKitchenRequest
import com.wanli.android.data.remote.UpdateRestrictionsRequest
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class KitchenRepository @Inject constructor(
    private val apiService: ApiService
) {
    suspend fun getKitchen(): ApiResult<KitchenData> {
        return safeCall {
            val response = apiService.getKitchen()
            if (response.success && response.data != null) {
                ApiResult.Success(response.data)
            } else {
                ApiResult.Error(
                    code = response.error?.code ?: "UNKNOWN_ERROR",
                    message = response.error?.message ?: "未知错误"
                )
            }
        }
    }

    suspend fun updateKitchen(ingredients: List<String>, tools: List<String>): ApiResult<KitchenData> {
        return safeCall {
            val response = apiService.updateKitchen(UpdateKitchenRequest(ingredients, tools))
            if (response.success && response.data != null) {
                ApiResult.Success(response.data)
            } else {
                ApiResult.Error(
                    code = response.error?.code ?: "UNKNOWN_ERROR",
                    message = response.error?.message ?: "未知错误"
                )
            }
        }
    }

    suspend fun getRestrictions(): ApiResult<List<RestrictionDto>> {
        return safeCall {
            val response = apiService.getDietaryRestrictions()
            if (response.success) {
                ApiResult.Success(response.data ?: emptyList())
            } else {
                ApiResult.Error(
                    code = response.error?.code ?: "UNKNOWN_ERROR",
                    message = response.error?.message ?: "未知错误"
                )
            }
        }
    }

    suspend fun updateRestrictions(restrictions: List<RestrictionDto>): ApiResult<List<RestrictionDto>> {
        return safeCall {
            val response = apiService.updateDietaryRestrictions(UpdateRestrictionsRequest(restrictions))
            if (response.success) {
                ApiResult.Success(response.data ?: emptyList())
            } else {
                ApiResult.Error(
                    code = response.error?.code ?: "UNKNOWN_ERROR",
                    message = response.error?.message ?: "未知错误"
                )
            }
        }
    }

    private inline fun <T> safeCall(call: () -> ApiResult<T>): ApiResult<T> {
        return try {
            call()
        } catch (e: Exception) {
            ApiResult.Error(
                code = "NETWORK_ERROR",
                message = e.message ?: "网络请求失败"
            )
        }
    }
}
