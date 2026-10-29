package com.wanli.android.presentation.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.local.SavedCredentials
import com.wanli.android.data.local.TokenDataStore
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.repository.AuthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    private val tokenDataStore: TokenDataStore
) : ViewModel() {

    private val _uiState = MutableStateFlow<LoginUiState>(LoginUiState.Idle)
    val uiState: StateFlow<LoginUiState> = _uiState

    /** 上次「记住密码」保存的账号密码，用于登录页自动填充。 */
    val savedCredentials: Flow<SavedCredentials?> = tokenDataStore.savedCredentials

    fun register(email: String, password: String, remember: Boolean) {
        submit(remember, email, password) {
            authRepository.register(sanitizeEmail(email), password)
        }
    }

    fun login(email: String, password: String, remember: Boolean) {
        submit(remember, email, password) {
            authRepository.login(sanitizeEmail(email), password)
        }
    }

    private fun <T> submit(
        remember: Boolean,
        email: String,
        password: String,
        request: suspend () -> ApiResult<T>
    ) {
        viewModelScope.launch {
            _uiState.value = LoginUiState.Loading
            _uiState.value = when (val result = request()) {
                is ApiResult.Success -> {
                    // 先落盘再切状态：页面随登录成功出栈，晚一步就可能被取消
                    val normalized = sanitizeEmail(email)
                    if (remember) {
                        tokenDataStore.saveCredentials(normalized, password)
                    } else {
                        tokenDataStore.clearCredentials()
                    }
                    LoginUiState.LoggedIn
                }
                is ApiResult.Error -> LoginUiState.Error(result.message)
                is ApiResult.Loading -> LoginUiState.Loading
            }
        }
    }
}

// sanitizeEmail 去除首尾空白与控制字符（0x00-0x1F、0x7F），
// 防止输入法/自动填充混入的不可见字符导致后端邮箱格式校验失败。
private fun sanitizeEmail(email: String): String =
    email.trim().filterNot { it.code < 0x20 || it.code == 0x7f }

sealed class LoginUiState {
    data object Idle : LoginUiState()
    data object Loading : LoginUiState()
    data object LoggedIn : LoginUiState()
    data class Error(val message: String) : LoginUiState()
}
