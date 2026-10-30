package com.wanli.android.presentation.update

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.BuildConfig
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.repository.AppUpdateRepository
import com.wanli.android.domain.model.AppUpdateInfo
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import java.io.File
import javax.inject.Inject

sealed interface AppUpdateUiState {
    /** 无弹窗。 */
    data object Idle : AppUpdateUiState

    /** 手动检查更新中。 */
    data object Checking : AppUpdateUiState

    /** 手动检查后确认已是最新。 */
    data class UpToDate(val currentVersion: String) : AppUpdateUiState

    data class Available(val info: AppUpdateInfo) : AppUpdateUiState

    data class Downloading(
        val info: AppUpdateInfo,
        val progress: Float,
        val downloaded: Long,
        val total: Long,
    ) : AppUpdateUiState

    data class Ready(val info: AppUpdateInfo, val apk: File) : AppUpdateUiState

    data class Failed(val info: AppUpdateInfo?, val message: String) : AppUpdateUiState
}

@HiltViewModel
class AppUpdateViewModel @Inject constructor(
    private val repository: AppUpdateRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow<AppUpdateUiState>(AppUpdateUiState.Idle)
    val uiState: StateFlow<AppUpdateUiState> = _uiState.asStateFlow()

    private var checkJob: Job? = null
    private var downloadJob: Job? = null
    private var autoChecked = false

    /** 启动时静默检查一次；失败或被忽略都不打扰用户。 */
    fun checkOnLaunch() {
        if (autoChecked) return
        autoChecked = true
        check(manual = false)
    }

    /**
     * @param manual       用户主动点击「检查更新」：显示检查中/已是最新/失败状态。
     *                     自动检查则只在发现新版本时弹窗。
     * @param autoDownload 发现新版本后立即开始下载（「我的 → 检查更新」用），
     *                     用户不必再点一次「立即更新」；安装仍需系统确认。
     */
    fun check(manual: Boolean, autoDownload: Boolean = false) {
        if (checkJob?.isActive == true) return
        checkJob = viewModelScope.launch {
            if (manual) _uiState.value = AppUpdateUiState.Checking
            when (val result = repository.check()) {
                is ApiResult.Success -> {
                    val info = result.data
                    _uiState.value = when {
                        info == null && manual -> AppUpdateUiState.UpToDate(BuildConfig.VERSION_NAME)
                        info == null -> AppUpdateUiState.Idle
                        !manual && repository.wasDismissedRecently(info.versionCode) -> AppUpdateUiState.Idle
                        else -> AppUpdateUiState.Available(info)
                    }
                    if (autoDownload && _uiState.value is AppUpdateUiState.Available) {
                        download()
                    }
                }
                is ApiResult.Error ->
                    _uiState.value = if (manual) {
                        AppUpdateUiState.Failed(null, result.message)
                    } else {
                        AppUpdateUiState.Idle
                    }
                is ApiResult.Loading -> Unit
            }
        }
    }

    fun download() {
        if (downloadJob?.isActive == true) return
        val info = when (val state = _uiState.value) {
            is AppUpdateUiState.Available -> state.info
            is AppUpdateUiState.Failed -> state.info
            else -> null
        } ?: return

        downloadJob = viewModelScope.launch {
            _uiState.value = AppUpdateUiState.Downloading(info, progress = 0f, downloaded = 0, total = info.size)
            val result = repository.download(info) { downloaded, total ->
                val progress = if (total > 0) {
                    (downloaded.toFloat() / total).coerceIn(0f, 1f)
                } else {
                    0f
                }
                _uiState.value = AppUpdateUiState.Downloading(info, progress, downloaded, total)
            }
            _uiState.value = when (result) {
                is ApiResult.Success -> AppUpdateUiState.Ready(info, result.data)
                is ApiResult.Error -> AppUpdateUiState.Failed(info, result.message)
                is ApiResult.Loading -> _uiState.value
            }
        }
    }

    /** 关闭弹窗；对「发现新版本」记住 24 小时，下载中则取消下载。 */
    fun dismiss() {
        when (val state = _uiState.value) {
            is AppUpdateUiState.Available -> repository.markDismissed(state.info.versionCode)
            is AppUpdateUiState.Failed -> state.info?.let { repository.markDismissed(it.versionCode) }
            is AppUpdateUiState.Downloading -> downloadJob?.cancel()
            else -> Unit
        }
        _uiState.value = AppUpdateUiState.Idle
    }
}
