package com.wanli.android.data.remote

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.PATCH
import retrofit2.http.POST
import retrofit2.http.PUT
import retrofit2.http.Path
import retrofit2.http.Query

interface ApiService {
    @GET("health")
    suspend fun health(): HealthResponse

    @POST("auth/register")
    suspend fun register(@Body request: RegisterRequest): LoginResponse

    @POST("auth/login")
    suspend fun login(@Body request: LoginRequest): LoginResponse

    @GET("me")
    suspend fun me(): MeResponse

    @PUT("me/profile")
    suspend fun updateProfile(@Body request: UpdateProfileRequest): MeResponse

    @GET("feed/cooking-records")
    suspend fun getCookingFeed(@Query("limit") limit: Int): CookingFeedResponse

    @GET("me/messages")
    suspend fun getMessages(): MessagesResponse

    @GET("me/cooking-records")
    suspend fun getMyCookingRecords(@Query("limit") limit: Int): MyCookingRecordsResponse

    @GET("me/kitchen")
    suspend fun getKitchen(): KitchenResponse

    @PUT("me/kitchen")
    suspend fun updateKitchen(@Body request: UpdateKitchenRequest): KitchenResponse

    @GET("me/dietary-restrictions")
    suspend fun getDietaryRestrictions(): RestrictionsResponse

    @PUT("me/dietary-restrictions")
    suspend fun updateDietaryRestrictions(@Body request: UpdateRestrictionsRequest): RestrictionsResponse

    @POST("video-imports")
    suspend fun createVideoImport(@Body request: CreateVideoImportRequest): VideoImportResponse

    @GET("video-imports/{taskId}")
    suspend fun getVideoImport(@Path("taskId") taskId: String): VideoImportTaskResponse

    @GET("recipe-versions/{id}")
    suspend fun getRecipeVersion(@Path("id") id: String): RecipeVersionResponse

    @PUT("recipe-versions/{id}/draft")
    suspend fun updateDraft(
        @Path("id") id: String,
        @Body request: UpdateDraftRequest
    ): RecipeVersionResponse

    @POST("recipe-versions/{id}/publish")
    suspend fun publishVersion(
        @Path("id") id: String,
        @Body request: UpdateDraftRequest
    ): PublishResponse

    @POST("recipe-versions/{id}/forks")
    suspend fun forkVersion(
        @Path("id") id: String,
        @Body request: ForkVersionRequest
    ): RecipeVersionResponse

    @POST("recipe-versions/{id}/adaptations")
    suspend fun adaptVersion(@Path("id") id: String): RecipeVersionResponse

    @PATCH("recipe-versions/{id}/visibility")
    suspend fun setVersionVisibility(
        @Path("id") id: String,
        @Body request: UpdateVisibilityRequest
    ): RecipeVersionResponse

    @GET("me/adaptations")
    suspend fun getMyAdaptations(): MyAdaptationsResponse

    @GET("recipe-versions/{id}/comments")
    suspend fun getComments(@Path("id") id: String): CommentsResponse

    @POST("recipe-versions/{id}/comments")
    suspend fun createComment(
        @Path("id") id: String,
        @Body request: CreateCommentRequest
    ): CommentResponse

    @DELETE("comments/{id}")
    suspend fun deleteComment(@Path("id") id: String): BaseResponse

    @GET("recipe-versions/{id}/cooking-records")
    suspend fun getCookingRecords(@Path("id") id: String): CookingRecordsResponse

    @POST("recipe-versions/{id}/cooking-records")
    suspend fun createCookingRecord(
        @Path("id") id: String,
        @Body request: CreateCookingRecordRequest
    ): CookingRecordResponse

    @DELETE("cooking-records/{id}")
    suspend fun deleteCookingRecord(@Path("id") id: String): BaseResponse

    @POST("upload/presigned-url")
    suspend fun presignUpload(@Body request: PresignUploadRequest): PresignUploadResponse

    @POST("recipe-versions/{id}/corrections")
    suspend fun submitCorrection(
        @Path("id") id: String,
        @Body request: SubmitCorrectionRequest
    ): CorrectionProposalResponse

    @GET("recipes/{id}/version-tree")
    suspend fun getVersionTree(@Path("id") id: String): VersionTreeResponse

    @GET("me/correction-proposals")
    suspend fun getPendingCorrections(): PendingCorrectionsResponse

    @POST("correction-proposals/{id}/review")
    suspend fun reviewCorrection(
        @Path("id") id: String,
        @Body request: ReviewCorrectionRequest
    ): ReviewCorrectionResponse

    @GET("recipes")
    suspend fun getRecipes(
        @Query("sort") sort: String,
        @Query("page") page: Int,
        @Query("page_size") pageSize: Int
    ): RecipeListResponse
}

@Serializable
data class BaseResponse(
    val success: Boolean,
    val data: kotlinx.serialization.json.JsonObject? = null,
    val error: ErrorBody? = null
)

@Serializable
data class HealthResponse(
    val success: Boolean,
    val data: HealthData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class HealthData(
    val status: String
)

@Serializable
data class ErrorBody(
    val code: String,
    val message: String,
    val details: kotlinx.serialization.json.JsonObject? = null
)

@Serializable
data class LoginRequest(
    val email: String,
    val password: String,
    @SerialName("turnstile_token") val turnstileToken: String
)

@Serializable
data class RegisterRequest(
    val email: String,
    val password: String,
    @SerialName("turnstile_token") val turnstileToken: String
)

@Serializable
data class LoginResponse(
    val success: Boolean,
    val data: LoginData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class LoginData(
    val token: String,
    val user: UserDto
)

@Serializable
data class UserDto(
    val id: String,
    val email: String,
    val nickname: String = "",
    val avatar_url: String? = null
)

@Serializable
data class MeResponse(
    val success: Boolean,
    val data: UserDto? = null,
    val error: ErrorBody? = null
)

@Serializable
data class UpdateProfileRequest(
    val nickname: String,
    @SerialName("avatar_url") val avatarUrl: String? = null,
)

@Serializable
data class CookingFeedItemDto(
    val id: String,
    val rating: Int? = null,
    @SerialName("photo_url") val photoUrl: String? = null,
    val feedback: String? = null,
    @SerialName("author_nickname") val authorNickname: String = "",
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("recipe_version_id") val recipeVersionId: String,
    @SerialName("recipe_id") val recipeId: String,
    @SerialName("recipe_title") val recipeTitle: String,
    @SerialName("cover_url") val coverUrl: String? = null,
)

@Serializable
data class CookingFeedData(val items: List<CookingFeedItemDto> = emptyList())

@Serializable
data class CookingFeedResponse(
    val success: Boolean,
    val data: CookingFeedData? = null,
    val error: ErrorBody? = null,
)

@Serializable
data class MessageDto(
    val type: String,
    val id: String,
    @SerialName("actor_nickname") val actorNickname: String = "",
    val text: String? = null,
    val rating: Int? = null,
    @SerialName("recipe_version_id") val recipeVersionId: String? = null,
    @SerialName("recipe_title") val recipeTitle: String = "",
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class MessagesData(val items: List<MessageDto> = emptyList())

@Serializable
data class MessagesResponse(
    val success: Boolean,
    val data: MessagesData? = null,
    val error: ErrorBody? = null,
)

@Serializable
data class MyCookingRecordDto(
    val id: String,
    val rating: Int? = null,
    @SerialName("photo_url") val photoUrl: String? = null,
    val feedback: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
    @SerialName("recipe_version_id") val recipeVersionId: String,
    @SerialName("recipe_title") val recipeTitle: String,
    @SerialName("cover_url") val coverUrl: String? = null,
)

@Serializable
data class MyCookingRecordsData(val items: List<MyCookingRecordDto> = emptyList())

@Serializable
data class MyCookingRecordsResponse(
    val success: Boolean,
    val data: MyCookingRecordsData? = null,
    val error: ErrorBody? = null,
)

@Serializable
data class KitchenResponse(
    val success: Boolean,
    val data: KitchenData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class KitchenData(
    val owned_ingredients: List<String> = emptyList(),
    val owned_tools: List<String> = emptyList()
)

@Serializable
data class UpdateKitchenRequest(
    val owned_ingredients: List<String>,
    val owned_tools: List<String>
)

@Serializable
data class RestrictionsResponse(
    val success: Boolean,
    val data: List<RestrictionDto>? = null,
    val error: ErrorBody? = null
)

@Serializable
data class RestrictionDto(
    val id: String? = null,
    val restriction_type: String,
    val ingredient_name: String
)

@Serializable
data class UpdateRestrictionsRequest(
    val restrictions: List<RestrictionDto>
)

@Serializable
data class CreateVideoImportRequest(
    val url: String
)

@Serializable
data class VideoImportResponse(
    val success: Boolean,
    val data: VideoImportData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class VideoImportData(
    val task_id: String,
    val status: String,
    val bvid: String
)

@Serializable
data class VideoImportTaskResponse(
    val success: Boolean,
    val data: VideoImportTaskData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class VideoImportTaskData(
    val task_id: String,
    val status: String,
    val bvid: String,
    val message: String? = null,
    val error_code: String? = null,
    val recipe_version_id: String? = null
)

@Serializable
data class RecipeVersionResponse(
    val success: Boolean,
    val data: RecipeVersionData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class RecipeVersionData(
    val id: String,
    val recipe_id: String,
    val parent_id: String? = null,
    val title: String,
    val up_nickname: String = "",
    val video_url: String? = null,
    val video_status: String = "ACTIVE",
    val video_warning: String? = null,
    val cover_url: String? = null,
    val version_type: String,
    val is_public: Boolean,
    val author_nickname: String = "",
    val change_log: String? = null,
    val servings: String? = null,
    val estimated_minutes: Int? = null,
    val instructions: String? = null,
    val adaptation_notes: AdaptationNotesDto? = null,
    val ingredients: List<IngredientDto> = emptyList(),
    val steps: List<StepDto> = emptyList()
)

@Serializable
data class IngredientDto(
    val name: String,
    val amount: String? = null,
    val category: String,
    val is_explicit: Boolean = true,
    val substitute_for: String? = null
)

@Serializable
data class StepDto(
    val step_order: Int,
    val description: String,
    val video_timestamp: Int? = null,
    val duration_seconds: Int? = null,
    val heat: String? = null
)

@Serializable
data class AdaptationNotesDto(
    val missing_ingredients: List<String> = emptyList(),
    val missing_tools: List<String> = emptyList(),
    val substitutions: List<AdaptationSubstitutionDto> = emptyList(),
    val flavor_changes: List<String> = emptyList()
)

@Serializable
data class AdaptationSubstitutionDto(
    val original: String,
    val replacement: String,
    val reason: String
)

@Serializable
data class UpdateVisibilityRequest(
    val is_public: Boolean
)

@Serializable
data class MyAdaptationsResponse(
    val success: Boolean,
    val data: MyAdaptationsData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class MyAdaptationsData(
    val items: List<MyAdaptationDto> = emptyList()
)

@Serializable
data class MyAdaptationDto(
    val id: String,
    val recipe_id: String,
    val parent_id: String? = null,
    val title: String,
    val is_public: Boolean,
    val created_at: String? = null
)

@Serializable
data class CommentsResponse(
    val success: Boolean,
    val data: CommentsData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class CommentsData(
    val items: List<CommentDto> = emptyList()
)

@Serializable
data class CommentResponse(
    val success: Boolean,
    val data: CommentDto? = null,
    val error: ErrorBody? = null
)

@Serializable
data class CommentDto(
    val id: String,
    val content: String,
    val author_nickname: String = "",
    val is_mine: Boolean = false,
    val created_at: String? = null
)

@Serializable
data class CreateCommentRequest(
    val content: String
)

@Serializable
data class CookingRecordsResponse(
    val success: Boolean,
    val data: CookingRecordsData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class CookingRecordsData(
    val items: List<CookingRecordDto> = emptyList()
)

@Serializable
data class CookingRecordResponse(
    val success: Boolean,
    val data: CookingRecordDto? = null,
    val error: ErrorBody? = null
)

@Serializable
data class CookingRecordDto(
    val id: String,
    val rating: Int? = null,
    val photo_url: String? = null,
    val feedback: String? = null,
    val author_nickname: String = "",
    val is_mine: Boolean = false,
    val created_at: String? = null
)

@Serializable
data class CreateCookingRecordRequest(
    val rating: Int? = null,
    val photo_url: String? = null,
    val feedback: String? = null
)

@Serializable
data class PresignUploadRequest(
    val filename: String,
    val content_type: String,
    val size_bytes: Long
)

@Serializable
data class PresignUploadResponse(
    val success: Boolean,
    val data: PresignUploadData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class PresignUploadData(
    val upload_url: String,
    val public_url: String,
    val object_name: String,
    val expires_in: Int
)

@Serializable
data class UpdateDraftRequest(
    val title: String,
    val servings: String? = null,
    val estimated_minutes: Int? = null,
    val instructions: String? = null,
    val ingredients: List<IngredientDto>,
    val steps: List<StepDto>
)

@Serializable
data class PublishResponse(
    val success: Boolean,
    val data: PublishData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class PublishData(
    val recipe_version_id: String,
    val is_public: Boolean
)

@Serializable
data class ForkVersionRequest(
    val title: String,
    val change_log: String,
    val servings: String? = null,
    val estimated_minutes: Int? = null,
    val instructions: String? = null,
    val ingredients: List<IngredientDto>,
    val steps: List<StepDto>
)

@Serializable
data class SubmitCorrectionRequest(
    val suggested_changes: UpdateDraftRequest,
    val reason: String
)

@Serializable
data class CorrectionProposalResponse(
    val success: Boolean,
    val data: CorrectionProposalData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class CorrectionProposalData(
    val proposal_id: String,
    val status: String
)

@Serializable
data class VersionTreeResponse(
    val success: Boolean,
    val data: VersionTreeData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class VersionTreeData(
    val recipe_id: String,
    val nodes: List<VersionTreeNodeDto> = emptyList()
)

@Serializable
data class VersionTreeNodeDto(
    val id: String,
    val title: String,
    val version_type: String,
    val parent_id: String? = null,
    val author_nickname: String = "",
    val change_log: String? = null,
    val created_at: String? = null
)

@Serializable
data class PendingCorrectionsResponse(
    val success: Boolean,
    val data: PendingCorrectionsData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class PendingCorrectionsData(
    val items: List<PendingCorrectionDto> = emptyList()
)

@Serializable
data class PendingCorrectionDto(
    val id: String,
    val recipe_version_id: String,
    val recipe_title: String,
    val proposer_nickname: String = "",
    val reason: String,
    val suggested_changes: UpdateDraftRequest,
    val created_at: String? = null
)

@Serializable
data class ReviewCorrectionRequest(
    val action: String,
    val rejection_reason: String = ""
)

@Serializable
data class ReviewCorrectionResponse(
    val success: Boolean,
    val data: ReviewCorrectionData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class ReviewCorrectionData(
    val action: String,
    val recipe_version_id: String? = null
)

@Serializable
data class RecipeListResponse(
    val success: Boolean,
    val data: RecipeListData? = null,
    val error: ErrorBody? = null
)

@Serializable
data class RecipeListData(
    val items: List<RecipeSummaryDto> = emptyList(),
    val total: Int = 0,
    val page: Int = 1,
    val page_size: Int = 20
)

@Serializable
data class RecipeSummaryDto(
    val recipe_id: String,
    val title: String,
    val cover_url: String? = null,
    val up_nickname: String = "",
    val video_status: String = "ACTIVE",
    val recipe_version_id: String,
    val servings: String? = null,
    val estimated_minutes: Int? = null,
    val author_nickname: String = "",
    val public_version_count: Int = 1,
    @SerialName("cooking_record_count") val cookingRecordCount: Int = 0,
    val created_at: String? = null
)
