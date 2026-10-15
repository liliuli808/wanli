package com.wanli.android.data.remote

import kotlinx.serialization.json.Json
import kotlinx.serialization.json.JsonObject
import retrofit2.HttpException

/** 后端统一错误包装（success/error 结构）还原出的业务错误。 */
data class BaseResponseError(
    val code: String,
    val message: String,
    val details: JsonObject? = null
)

private val errorJson = Json { ignoreUnknownKeys = true }

/** 非 2xx 响应时，从 errorBody 还原后端统一错误包装中的业务错误码与文案。 */
fun HttpException.parseBaseError(): BaseResponseError? {
    val raw = response()?.errorBody()?.string() ?: return null
    return try {
        val parsed = errorJson.decodeFromString(BaseResponse.serializer(), raw)
        parsed.error?.let { BaseResponseError(it.code, it.message, it.details) }
    } catch (_: Exception) {
        null
    }
}
