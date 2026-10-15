package com.wanli.android.data.repository

import com.wanli.android.data.local.TokenDataStore
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.LoginRequest
import com.wanli.android.data.remote.RegisterRequest
import com.wanli.android.data.remote.UpdateProfileRequest
import com.wanli.android.data.remote.UserDto
import com.wanli.android.data.remote.parseBaseError
import retrofit2.HttpException
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class AuthRepository @Inject constructor(
    private val apiService: ApiService,
    private val tokenDataStore: TokenDataStore
) {
    suspend fun register(email: String, password: String, turnstileToken: String): ApiResult<UserDto> = wrap(
        fallbackMessage = "注册失败"
    ) {
        val response = apiService.register(RegisterRequest(email, password, turnstileToken))
        if (response.success && response.data != null) {
            tokenDataStore.saveToken(
                token = response.data.token,
                userId = response.data.user.id,
                email = response.data.user.email
            )
            response.data.user
        } else {
            null
        }
    }

    suspend fun login(email: String, password: String, turnstileToken: String): ApiResult<UserDto> = wrap(
        fallbackMessage = "登录失败"
    ) {
        val response = apiService.login(LoginRequest(email, password, turnstileToken))
        if (response.success && response.data != null) {
            tokenDataStore.saveToken(
                token = response.data.token,
                userId = response.data.user.id,
                email = response.data.user.email
            )
            response.data.user
        } else {
            null
        }
    }

    suspend fun logout() {
        tokenDataStore.clearToken()
    }

    suspend fun getMe(): ApiResult<UserDto> = wrap(
        fallbackMessage = "加载用户信息失败"
    ) {
        val response = apiService.me()
        response.data?.takeIf { response.success }
    }

    suspend fun updateProfile(nickname: String, avatarUrl: String?): ApiResult<UserDto> = wrap(
        fallbackMessage = "更新资料失败"
    ) {
        val response = apiService.updateProfile(UpdateProfileRequest(nickname, avatarUrl))
        response.data?.takeIf { response.success }
    }

    val isLoggedIn = tokenDataStore.isLoggedIn

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
