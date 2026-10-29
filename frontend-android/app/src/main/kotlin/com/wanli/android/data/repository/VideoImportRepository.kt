package com.wanli.android.data.repository

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.CreateVideoImportRequest
import com.wanli.android.data.remote.VideoImportData
import com.wanli.android.data.remote.VideoImportTaskData
import com.wanli.android.data.remote.parseBaseError
import kotlinx.serialization.json.JsonObject
import kotlinx.serialization.json.jsonPrimitive
import retrofit2.HttpException
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class VideoImportRepository @Inject constructor(
    private val apiService: ApiService
) {
    sealed class SubmitResult {
        data class Success(val data: VideoImportData) : SubmitResult()
        data class Duplicate(val recipeId: String?, val recipeVersionId: String?, val message: String) : SubmitResult()
        data class Error(val code: String, val message: String) : SubmitResult()
    }

    suspend fun submit(url: String): SubmitResult {
        return try {
            val response = apiService.createVideoImport(CreateVideoImportRequest(url))
            val data = response.data
            if (response.success && data != null) {
                SubmitResult.Success(data)
            } else {
                SubmitResult.Error(
                    code = response.error?.code ?: "UNKNOWN_ERROR",
                    message = response.error?.message ?: "未知错误"
                )
            }
        } catch (e: HttpException) {
            // 409 BVID_ALREADY_PARSED：重复视频，提示并跳转已有菜谱
            val errorBody = e.parseBaseError()
            if (errorBody?.code == "BVID_ALREADY_PARSED") {
                val details = errorBody.details as? JsonObject
                SubmitResult.Duplicate(
                    recipeId = details?.get("recipe_id")?.jsonPrimitive?.content,
                    recipeVersionId = details?.get("recipe_version_id")?.jsonPrimitive?.content,
                    message = errorBody.message
                )
            } else {
                SubmitResult.Error(
                    code = errorBody?.code ?: "HTTP_${e.code()}",
                    // details 里是服务端给出的具体原因（如「请输入有效的B站视频链接」），优先展示
                    message = errorBody?.detailText
                        ?: errorBody?.message
                        ?: "提交失败，请稍后再试"
                )
            }
        } catch (e: Exception) {
            SubmitResult.Error(code = "NETWORK_ERROR", message = e.message ?: "网络请求失败")
        }
    }

    suspend fun getTask(taskId: String): ApiResult<VideoImportTaskData> {
        return try {
            val response = apiService.getVideoImport(taskId)
            val data = response.data
            if (response.success && data != null) {
                ApiResult.Success(data)
            } else {
                ApiResult.Error(
                    code = response.error?.code ?: "UNKNOWN_ERROR",
                    message = response.error?.message ?: "未知错误"
                )
            }
        } catch (e: HttpException) {
            val errorBody = e.parseBaseError()
            ApiResult.Error(
                code = errorBody?.code ?: "HTTP_${e.code()}",
                message = errorBody?.message ?: "查询任务失败"
            )
        } catch (e: Exception) {
            ApiResult.Error(code = "NETWORK_ERROR", message = e.message ?: "网络请求失败")
        }
    }
}
