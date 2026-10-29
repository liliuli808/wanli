package com.wanli.android.presentation.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.repository.BilibiliCredentialRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

data class BilibiliAuthUiState(
    val loading: Boolean = true,
    val loadError: String? = null,
    val bound: Boolean = false,
    val uname: String = "",
    val mid: String = "",
    /** 正在提交 SESSDATA。 */
    val binding: Boolean = false,
    /** 一次性提示（错误或成功）。 */
    val message: String? = null,
    /** 授权成功，界面据此返回上一页。 */
    val bindSucceeded: Boolean = false,
)

@HiltViewModel
class BilibiliAuthViewModel @Inject constructor(
    private val repository: BilibiliCredentialRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow(BilibiliAuthUiState())
    val uiState: StateFlow<BilibiliAuthUiState> = _uiState.asStateFlow()

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(loading = true, loadError = null)
            when (val result = repository.status()) {
                is ApiResult.Success -> _uiState.value = _uiState.value.copy(
                    loading = false,
                    bound = result.data.bound,
                    uname = result.data.uname,
                    mid = result.data.mid,
                )
                is ApiResult.Error -> _uiState.value = _uiState.value.copy(
                    loading = false,
                    loadError = result.message,
                )
                is ApiResult.Loading -> Unit
            }
        }
    }

    /** 提交从 WebView 或手动粘贴拿到的 SESSDATA。 */
    fun bind(sessdata: String) {
        val trimmed = sessdata.trim()
        if (trimmed.isEmpty() || _uiState.value.binding) return

        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(binding = true, message = null)
            when (val result = repository.bind(trimmed)) {
                is ApiResult.Success -> _uiState.value = _uiState.value.copy(
                    binding = false,
                    bound = result.data.bound,
                    uname = result.data.uname,
                    mid = result.data.mid,
                    bindSucceeded = true,
                    message = "授权成功，现在可以解析仅登录可见的字幕了",
                )
                is ApiResult.Error -> _uiState.value = _uiState.value.copy(
                    binding = false,
                    message = result.message,
                )
                is ApiResult.Loading -> Unit
            }
        }
    }

    fun unbind() {
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(binding = true, message = null)
            when (val result = repository.unbind()) {
                is ApiResult.Success -> _uiState.value = _uiState.value.copy(
                    binding = false,
                    bound = false,
                    uname = "",
                    mid = "",
                    message = "已解除授权",
                )
                is ApiResult.Error -> _uiState.value = _uiState.value.copy(
                    binding = false,
                    message = result.message,
                )
                is ApiResult.Loading -> Unit
            }
        }
    }

    fun clearMessage() {
        _uiState.value = _uiState.value.copy(message = null)
    }
}
