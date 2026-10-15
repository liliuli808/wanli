package com.wanli.android.data.repository

import android.content.ContentResolver
import android.content.Context
import android.net.Uri
import android.provider.OpenableColumns
import com.wanli.android.data.remote.ApiResult
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import okhttp3.MediaType
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody
import okio.BufferedSink
import okio.source
import java.io.IOException
import javax.inject.Inject
import javax.inject.Singleton

@Singleton
class PhotoUploadRepository @Inject constructor(
    @ApplicationContext private val context: Context,
    private val recipeRepository: RecipeRepository
) {
    private val uploadClient = OkHttpClient()

    suspend fun upload(uri: Uri): ApiResult<String> = withContext(Dispatchers.IO) {
        val metadata = readMetadata(uri)
            ?: return@withContext ApiResult.Error("VALIDATION_ERROR", "无法读取所选图片")
        if (metadata.contentType !in SUPPORTED_TYPES) {
            return@withContext ApiResult.Error("VALIDATION_ERROR", "仅支持 JPEG、PNG、WebP 图片")
        }
        if (metadata.size !in 1..MAX_PHOTO_BYTES) {
            return@withContext ApiResult.Error("VALIDATION_ERROR", "成品图大小须在 10MB 以内")
        }

        when (val presign = recipeRepository.presignUpload(
            filename = metadata.filename,
            contentType = metadata.contentType,
            sizeBytes = metadata.size
        )) {
            is ApiResult.Success -> {
                try {
                    val request = Request.Builder()
                        .url(presign.data.upload_url)
                        .put(
                            ContentUriRequestBody(
                                resolver = context.contentResolver,
                                uri = uri,
                                mediaType = metadata.contentType.toMediaType(),
                                size = metadata.size
                            )
                        )
                        .build()
                    uploadClient.newCall(request).execute().use { response ->
                        if (response.isSuccessful) {
                            ApiResult.Success(presign.data.public_url)
                        } else {
                            ApiResult.Error(
                                code = "UPLOAD_FAILED",
                                message = "图片上传失败（${response.code}）"
                            )
                        }
                    }
                } catch (e: Exception) {
                    ApiResult.Error("UPLOAD_FAILED", e.message ?: "图片上传失败")
                }
            }
            is ApiResult.Error -> presign
            is ApiResult.Loading -> ApiResult.Loading
        }
    }

    private fun readMetadata(uri: Uri): PhotoMetadata? {
        val resolver = context.contentResolver
        val contentType = resolver.getType(uri) ?: return null
        var filename = "cooking-photo"
        var size = -1L
        resolver.query(
            uri,
            arrayOf(OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE),
            null,
            null,
            null
        )?.use { cursor ->
            if (cursor.moveToFirst()) {
                val nameIndex = cursor.getColumnIndex(OpenableColumns.DISPLAY_NAME)
                val sizeIndex = cursor.getColumnIndex(OpenableColumns.SIZE)
                if (nameIndex >= 0) filename = cursor.getString(nameIndex) ?: filename
                if (sizeIndex >= 0 && !cursor.isNull(sizeIndex)) size = cursor.getLong(sizeIndex)
            }
        }
        return PhotoMetadata(filename, contentType, size)
    }

    private data class PhotoMetadata(
        val filename: String,
        val contentType: String,
        val size: Long
    )

    private class ContentUriRequestBody(
        private val resolver: ContentResolver,
        private val uri: Uri,
        private val mediaType: MediaType,
        private val size: Long
    ) : RequestBody() {
        override fun contentType(): MediaType = mediaType

        override fun contentLength(): Long = size

        override fun writeTo(sink: BufferedSink) {
            val stream = resolver.openInputStream(uri)
                ?: throw IOException("无法打开所选图片")
            stream.source().use { source -> sink.writeAll(source) }
        }
    }

    companion object {
        const val MAX_PHOTO_BYTES = 10L * 1024 * 1024
        val SUPPORTED_TYPES = setOf("image/jpeg", "image/png", "image/webp")
    }
}
