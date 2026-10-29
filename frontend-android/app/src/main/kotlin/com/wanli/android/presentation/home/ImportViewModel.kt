package com.wanli.android.presentation.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.repository.VideoImportRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.Job
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class ImportViewModel @Inject constructor(
    private val videoImportRepository: VideoImportRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow<ImportUiState>(ImportUiState.Idle)
    val uiState: StateFlow<ImportUiState> = _uiState

    private var pollingJob: Job? = null

    fun submit(url: String) {
        val trimmed = url.trim()
        if (trimmed.isEmpty()) {
            _uiState.value = ImportUiState.Failed("请先粘贴 B 站视频链接", errorCode = null)
            return
        }

        pollingJob?.cancel()
        _uiState.value = ImportUiState.Submitting

        viewModelScope.launch {
            when (val result = videoImportRepository.submit(trimmed)) {
                is VideoImportRepository.SubmitResult.Success -> startPolling(result.data.task_id)
                is VideoImportRepository.SubmitResult.Duplicate -> {
                    // AC1：重复 BVID，提示并引导跳转已有菜谱
                    _uiState.value = ImportUiState.Duplicate(
                        recipeId = result.recipeId,
                        recipeVersionId = result.recipeVersionId,
                        message = result.message
                    )
                }
                is VideoImportRepository.SubmitResult.Error -> {
                    _uiState.value = if (result.code == "UNAUTHORIZED") {
                        ImportUiState.LoginRequired
                    } else {
                        ImportUiState.Failed(result.message, result.code)
                    }
                }
            }
        }
    }

    private fun startPolling(taskId: String) {
        _uiState.value = ImportUiState.Polling("任务已创建，等待解析")
        pollingJob = viewModelScope.launch {
            repeat(MAX_POLL_COUNT) {
                delay(POLL_INTERVAL_MS)
                when (val result = videoImportRepository.getTask(taskId)) {
                    is ApiResult.Success -> {
                        val task = result.data
                        when (task.status) {
                            "COMPLETED" -> {
                                _uiState.value = ImportUiState.Completed(task.recipe_version_id)
                                return@launch
                            }
                            "FAILED" -> {
                                _uiState.value = ImportUiState.Failed(
                                    message = failureMessage(task.message, task.error_code),
                                    errorCode = task.error_code
                                )
                                return@launch
                            }
                            else -> {
                                _uiState.value = ImportUiState.Polling(
                                    task.message ?: defaultProgressText(task.status)
                                )
                            }
                        }
                    }
                    is ApiResult.Error -> {
                        // 轮询中的瞬时网络错误不打断，继续等下一轮
                        if (result.code != "NETWORK_ERROR") {
                            _uiState.value = ImportUiState.Failed(result.message, result.code)
                            return@launch
                        }
                    }
                    is ApiResult.Loading -> Unit
                }
            }
            _uiState.value = ImportUiState.Failed("解析超时，请稍后重新提交", errorCode = null)
        }
    }

    /** 重置到初始状态（用户收起结果卡片或重新输入时调用）。 */
    fun reset() {
        pollingJob?.cancel()
        _uiState.value = ImportUiState.Idle
    }

    override fun onCleared() {
        pollingJob?.cancel()
        super.onCleared()
    }

    companion object {
        const val POLL_INTERVAL_MS = 1500L
        const val MAX_POLL_COUNT = 120 // 最长轮询 3 分钟，与后端单任务超时一致

        fun defaultProgressText(status: String): String = when (status) {
            "PENDING" -> "排队等待解析"
            "FETCHING" -> "正在获取视频信息"
            "PARSING" -> "正在用AI拆解菜谱"
            else -> "解析中"
        }

        /** 后端已给出用户可读文案时优先使用；否则按错误码兜底。 */
        fun failureMessage(message: String?, errorCode: String?): String {
            if (!message.isNullOrBlank()) return message
            return when (errorCode) {
                "SUBTITLE_NOT_AVAILABLE" -> "该视频没有可用字幕，无法解析"
                "BILIBILI_LOGIN_REQUIRED" -> "该视频的字幕需要登录 B 站才能读取，请先在「我的 - B 站账号」中授权"
                "AI_PARSE_FAILED" -> "AI 未能从视频中提取菜谱，可重试一次"
                "VIDEO_FETCH_FAILED" -> "获取视频信息失败，请检查链接或稍后再试"
                else -> "解析失败，请稍后再试"
            }
        }
    }
}

sealed class ImportUiState {
    data object Idle : ImportUiState()
    data object Submitting : ImportUiState()
    data class Polling(val message: String) : ImportUiState()
    data class Completed(val recipeVersionId: String?) : ImportUiState()
    data class Failed(val message: String, val errorCode: String?) : ImportUiState()
    data class Duplicate(val recipeId: String?, val recipeVersionId: String?, val message: String) : ImportUiState()
    data object LoginRequired : ImportUiState()
}
