package com.wanli.android.data.repository

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.CookingFeedItemDto
import com.wanli.android.data.remote.MessageDto
import com.wanli.android.data.remote.MyCookingRecordDto
import com.wanli.android.data.remote.parseBaseError
import retrofit2.HttpException
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class SocialRepository @Inject constructor(
    private val apiService: ApiService
) {
    suspend fun getCookingFeed(limit: Int = 20): ApiResult<List<CookingFeedItemDto>> = wrap(
        fallbackMessage = "加载动态失败"
    ) {
        val response = apiService.getCookingFeed(limit)
        response.data?.takeIf { response.success }?.items
    }

    suspend fun getMessages(): ApiResult<List<MessageDto>> = wrap(
        fallbackMessage = "加载消息失败"
    ) {
        val response = apiService.getMessages()
        response.data?.takeIf { response.success }?.items
    }

    suspend fun getMyCookingRecords(limit: Int = 60): ApiResult<List<MyCookingRecordDto>> = wrap(
        fallbackMessage = "加载做过失败"
    ) {
        val response = apiService.getMyCookingRecords(limit)
        response.data?.takeIf { response.success }?.items
    }

    private suspend fun <T> wrap(
        fallbackMessage: String,
        call: suspend () -> T?
    ): ApiResult<T> {
        return try {
            val data = call()
            if (data != null) {
                ApiResult.Success(data)
            } else {
                ApiResult.Error(code = "UNKNOWN_ERROR", message = fallbackMessage)
            }
        } catch (e: HttpException) {
            val errorBody = e.parseBaseError()
            ApiResult.Error(
                code = errorBody?.code ?: "HTTP_${e.code()}",
                message = errorBody?.message ?: fallbackMessage
            )
        } catch (e: Exception) {
            ApiResult.Error(code = "NETWORK_ERROR", message = e.message ?: "网络请求失败")
        }
    }
}
