package com.wanli.android.data.repository

import android.content.Context
import com.wanli.android.BuildConfig
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ApiService
import com.wanli.android.data.remote.parseBaseError
import com.wanli.android.domain.model.AppUpdateInfo
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.currentCoroutineContext
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.withContext
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.OkHttpClient
import okhttp3.Request
import retrofit2.HttpException
import java.io.File
import java.io.FileOutputStream
import java.security.MessageDigest
import java.util.concurrent.TimeUnit
import javax.inject.Inject
import javax.inject.Named
import javax.inject.Singleton

/**
 * App 自更新：检查版本、下载并校验安装包。
 *
 * 版本信息与 APK 都走自家后端代理（服务端再转发 GitHub Releases），
 * 避免国内网络直连 GitHub 失败。
 */
@Singleton
class AppUpdateRepository @Inject constructor(
    @ApplicationContext private val context: Context,
    private val apiService: ApiService,
    @Named("download") private val downloadClient: OkHttpClient,
    apiBaseUrl: String,
) {
    private val baseUrl = apiBaseUrl.toHttpUrl()
    private val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)

    /**
     * 查询最新版本；已是最新（或服务端未配置）时返回 null。
     */
    suspend fun check(): ApiResult<AppUpdateInfo?> {
        return try {
            val response = apiService.getAppUpdate()
            val data = response.data
            if (!response.success || data == null) {
                ApiResult.Success(null)
            } else if (data.versionCode <= BuildConfig.VERSION_CODE) {
                ApiResult.Success(null)
            } else {
                ApiResult.Success(
                    AppUpdateInfo(
                        versionCode = data.versionCode,
                        versionName = data.versionName,
                        size = data.size,
                        sha256 = data.sha256,
                        commit = data.commit,
                        downloadUrl = baseUrl.resolve(data.downloadPath)?.toString()
                            ?: return ApiResult.Error("UPDATE_BAD_URL", "版本信息中的下载地址无效"),
                    )
                )
            }
        } catch (e: HttpException) {
            val errorBody = e.parseBaseError()
            ApiResult.Error(
                code = errorBody?.code ?: "HTTP_${e.code()}",
                message = errorBody?.message ?: "检查更新失败",
            )
        } catch (e: Exception) {
            ApiResult.Error("NETWORK_ERROR", e.message ?: "检查更新失败")
        }
    }

    /**
     * 下载安装包并校验 SHA-256，返回可安装的本地文件。
     * [onProgress] 回调已下载字节数与总字节数（总长未知时为 0）。
     */
    suspend fun download(
        info: AppUpdateInfo,
        onProgress: (downloaded: Long, total: Long) -> Unit,
    ): ApiResult<File> = withContext(Dispatchers.IO) {
        val dir = File(context.cacheDir, UPDATE_DIR).apply { mkdirs() }
        // 只保留本次文件，避免历史安装包堆积
        dir.listFiles()?.forEach { it.delete() }

        val target = File(dir, "wanli-${info.versionCode}.apk")
        val part = File(dir, "wanli-${info.versionCode}.apk.part")

        try {
            val request = Request.Builder().url(info.downloadUrl).build()
            downloadClient.newCall(request).execute().use { response ->
                if (!response.isSuccessful) {
                    return@withContext ApiResult.Error(
                        code = "DOWNLOAD_FAILED",
                        message = "下载失败（HTTP ${response.code}）",
                    )
                }
                val body = response.body
                    ?: return@withContext ApiResult.Error("DOWNLOAD_FAILED", "下载失败：响应为空")

                val total = body.contentLength().takeIf { it > 0 } ?: info.size
                val digest = MessageDigest.getInstance("SHA-256")
                var downloaded = 0L
                var lastReported = 0L

                body.byteStream().use { input ->
                    FileOutputStream(part).use { output ->
                        val buffer = ByteArray(DEFAULT_BUFFER_SIZE)
                        while (true) {
                            currentCoroutineContext().ensureActive()
                            val read = input.read(buffer)
                            if (read < 0) break
                            output.write(buffer, 0, read)
                            digest.update(buffer, 0, read)
                            downloaded += read
                            if (downloaded - lastReported >= PROGRESS_STEP) {
                                lastReported = downloaded
                                onProgress(downloaded, total)
                            }
                        }
                        output.flush()
                        output.fd.sync()
                    }
                }
                onProgress(downloaded, total)

                if (info.sha256.isNotBlank()) {
                    val actual = digest.digest().joinToString("") { "%02x".format(it) }
                    if (!actual.equals(info.sha256, ignoreCase = true)) {
                        part.delete()
                        return@withContext ApiResult.Error(
                            code = "CHECKSUM_MISMATCH",
                            message = "安装包校验失败，请重试",
                        )
                    }
                }
            }

            if (!part.renameTo(target)) {
                part.delete()
                return@withContext ApiResult.Error("DOWNLOAD_FAILED", "保存安装包失败")
            }
            ApiResult.Success(target)
        } catch (e: Exception) {
            part.delete()
            if (e is kotlinx.coroutines.CancellationException) throw e
            ApiResult.Error("DOWNLOAD_FAILED", e.message ?: "下载失败")
        }
    }

    /** 同一版本被用户忽略后 24 小时内不再自动提示。 */
    fun wasDismissedRecently(versionCode: Int): Boolean {
        val code = prefs.getInt(KEY_DISMISSED_CODE, -1)
        val at = prefs.getLong(KEY_DISMISSED_AT, 0L)
        return code == versionCode && System.currentTimeMillis() - at < DISMISS_TTL_MS
    }

    fun markDismissed(versionCode: Int) {
        prefs.edit()
            .putInt(KEY_DISMISSED_CODE, versionCode)
            .putLong(KEY_DISMISSED_AT, System.currentTimeMillis())
            .apply()
    }

    private companion object {
        const val PREFS_NAME = "app_update"
        const val KEY_DISMISSED_CODE = "dismissed_version_code"
        const val KEY_DISMISSED_AT = "dismissed_at"
        const val UPDATE_DIR = "updates"
        const val PROGRESS_STEP = 256L * 1024
        val DISMISS_TTL_MS = TimeUnit.HOURS.toMillis(24)
    }
}
