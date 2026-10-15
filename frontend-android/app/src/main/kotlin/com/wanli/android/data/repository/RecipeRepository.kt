package com.wanli.android.data.repository

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.PublishData
import com.wanli.android.data.remote.CorrectionProposalData
import com.wanli.android.data.remote.CommentDto
import com.wanli.android.data.remote.CommentsData
import com.wanli.android.data.remote.CookingRecordDto
import com.wanli.android.data.remote.CookingRecordsData
import com.wanli.android.data.remote.CreateCommentRequest
import com.wanli.android.data.remote.CreateCookingRecordRequest
import com.wanli.android.data.remote.ForkVersionRequest
import com.wanli.android.data.remote.PendingCorrectionsData
import com.wanli.android.data.remote.MyAdaptationsData
import com.wanli.android.data.remote.RecipeListData
import com.wanli.android.data.remote.RecipeVersionData
import com.wanli.android.data.remote.ReviewCorrectionData
import com.wanli.android.data.remote.ReviewCorrectionRequest
import com.wanli.android.data.remote.PresignUploadData
import com.wanli.android.data.remote.PresignUploadRequest
import com.wanli.android.data.remote.SubmitCorrectionRequest
import com.wanli.android.data.remote.UpdateDraftRequest
import com.wanli.android.data.remote.UpdateVisibilityRequest
import com.wanli.android.data.remote.VersionTreeData
import com.wanli.android.data.remote.parseBaseError
import retrofit2.HttpException
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class RecipeRepository @Inject constructor(
    private val apiService: ApiService
) {
    suspend fun getVersionDetail(versionId: String): ApiResult<RecipeVersionData> = wrap(
        fallbackMessage = "加载菜谱失败"
    ) {
        val response = apiService.getRecipeVersion(versionId)
        response.data?.takeIf { response.success }
    }

    suspend fun updateDraft(versionId: String, request: UpdateDraftRequest): ApiResult<RecipeVersionData> = wrap(
        fallbackMessage = "保存草稿失败"
    ) {
        val response = apiService.updateDraft(versionId, request)
        response.data?.takeIf { response.success }
    }

    suspend fun publish(versionId: String, request: UpdateDraftRequest): ApiResult<PublishData> = wrap(
        fallbackMessage = "发布失败，请稍后再试"
    ) {
        val response = apiService.publishVersion(versionId, request)
        response.data?.takeIf { response.success }
    }

    suspend fun fork(versionId: String, request: ForkVersionRequest): ApiResult<RecipeVersionData> = wrap(
        fallbackMessage = "发布改编版失败"
    ) {
        val response = apiService.forkVersion(versionId, request)
        response.data?.takeIf { response.success }
    }

    suspend fun adapt(versionId: String): ApiResult<RecipeVersionData> = wrap(
        fallbackMessage = "厨房适配失败，请稍后重试"
    ) {
        val response = apiService.adaptVersion(versionId)
        response.data?.takeIf { response.success }
    }

    suspend fun setVisibility(versionId: String, isPublic: Boolean): ApiResult<RecipeVersionData> = wrap(
        fallbackMessage = "更新公开状态失败"
    ) {
        val response = apiService.setVersionVisibility(versionId, UpdateVisibilityRequest(isPublic))
        response.data?.takeIf { response.success }
    }

    suspend fun getMyAdaptations(): ApiResult<MyAdaptationsData> = wrap(
        fallbackMessage = "加载个人适配版失败"
    ) {
        val response = apiService.getMyAdaptations()
        response.data?.takeIf { response.success }
    }

    suspend fun getComments(versionId: String): ApiResult<CommentsData> = wrap(
        fallbackMessage = "加载评论失败"
    ) {
        val response = apiService.getComments(versionId)
        response.data?.takeIf { response.success }
    }

    suspend fun createComment(versionId: String, content: String): ApiResult<CommentDto> = wrap(
        fallbackMessage = "发表评论失败"
    ) {
        val response = apiService.createComment(versionId, CreateCommentRequest(content))
        response.data?.takeIf { response.success }
    }

    suspend fun deleteComment(commentId: String): ApiResult<Unit> = wrap(
        fallbackMessage = "删除评论失败"
    ) {
        val response = apiService.deleteComment(commentId)
        Unit.takeIf { response.success }
    }

    suspend fun getCookingRecords(versionId: String): ApiResult<CookingRecordsData> = wrap(
        fallbackMessage = "加载成品记录失败"
    ) {
        val response = apiService.getCookingRecords(versionId)
        response.data?.takeIf { response.success }
    }

    suspend fun createCookingRecord(
        versionId: String,
        request: CreateCookingRecordRequest
    ): ApiResult<CookingRecordDto> = wrap(fallbackMessage = "提交“做过”记录失败") {
        val response = apiService.createCookingRecord(versionId, request)
        response.data?.takeIf { response.success }
    }

    suspend fun deleteCookingRecord(recordId: String): ApiResult<Unit> = wrap(
        fallbackMessage = "删除“做过”记录失败"
    ) {
        val response = apiService.deleteCookingRecord(recordId)
        Unit.takeIf { response.success }
    }

    suspend fun presignUpload(
        filename: String,
        contentType: String,
        sizeBytes: Long
    ): ApiResult<PresignUploadData> = wrap(fallbackMessage = "获取图片上传地址失败") {
        val response = apiService.presignUpload(
            PresignUploadRequest(filename, contentType, sizeBytes)
        )
        response.data?.takeIf { response.success }
    }

    suspend fun submitCorrection(
        versionId: String,
        request: SubmitCorrectionRequest
    ): ApiResult<CorrectionProposalData> = wrap(fallbackMessage = "提交纠错失败") {
        val response = apiService.submitCorrection(versionId, request)
        response.data?.takeIf { response.success }
    }

    suspend fun getVersionTree(recipeId: String): ApiResult<VersionTreeData> = wrap(
        fallbackMessage = "加载版本树失败"
    ) {
        val response = apiService.getVersionTree(recipeId)
        response.data?.takeIf { response.success }
    }

    suspend fun getPendingCorrections(): ApiResult<PendingCorrectionsData> = wrap(
        fallbackMessage = "加载待审核纠错失败"
    ) {
        val response = apiService.getPendingCorrections()
        response.data?.takeIf { response.success }
    }

    suspend fun reviewCorrection(
        proposalId: String,
        request: ReviewCorrectionRequest
    ): ApiResult<ReviewCorrectionData> = wrap(fallbackMessage = "审核失败") {
        val response = apiService.reviewCorrection(proposalId, request)
        response.data?.takeIf { response.success }
    }

    suspend fun listRecipes(sort: String, page: Int, pageSize: Int): ApiResult<RecipeListData> = wrap(
        fallbackMessage = "加载菜谱列表失败"
    ) {
        val response = apiService.getRecipes(sort, page, pageSize)
        response.data?.takeIf { response.success }
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
