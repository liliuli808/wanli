package com.wanli.android.data.repository

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.BilibiliCredentialData
import com.wanli.android.data.remote.BindBilibiliCredentialRequest
import com.wanli.android.data.remote.parseBaseError
import retrofit2.HttpException
import javax.inject.Inject
import javax.inject.Singleton

/**
 * 用户级 B 站登录态授权。
 *
 * 这里只上传 SESSDATA，不上传完整 Cookie；服务端加密保存且不会回传任何凭证内容。
 */
@Singleton
class BilibiliCredentialRepository @Inject constructor(
    private val apiService: ApiService
) {
    suspend fun status(): ApiResult<BilibiliCredentialData> = wrap(
        fallbackMessage = "查询 B 站授权失败"
    ) {
        val response = apiService.getBilibiliCredential()
        response.data?.takeIf { response.success }
    }

    suspend fun bind(sessdata: String): ApiResult<BilibiliCredentialData> = wrap(
        fallbackMessage = "授权失败"
    ) {
        val response = apiService.bindBilibiliCredential(BindBilibiliCredentialRequest(sessdata))
        response.data?.takeIf { response.success }
    }

    suspend fun unbind(): ApiResult<Unit> = wrap(
        fallbackMessage = "解除授权失败"
    ) {
        val response = apiService.unbindBilibiliCredential()
        Unit.takeIf { response.success }
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
