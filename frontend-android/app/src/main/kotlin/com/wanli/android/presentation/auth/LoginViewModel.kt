package com.wanli.android.presentation.auth

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.repository.AuthRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject
import javax.inject.Named

@HiltViewModel
class LoginViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    @Named("serverRootUrl") private val serverRootUrl: String
) : ViewModel() {

    private val _uiState = MutableStateFlow<LoginUiState>(LoginUiState.Idle)
    val uiState: StateFlow<LoginUiState> = _uiState

    val captchaUrl: String
        get() = "$serverRootUrl/captcha"

    fun register(email: String, password: String, turnstileToken: String) {
        viewModelScope.launch {
            _uiState.value = LoginUiState.Loading
            _uiState.value = when (val result = authRepository.register(sanitizeEmail(email), password, turnstileToken)) {
                is ApiResult.Success -> LoginUiState.LoggedIn
                is ApiResult.Error -> LoginUiState.Error(result.message)
                is ApiResult.Loading -> LoginUiState.Loading
            }
        }
    }

    fun login(email: String, password: String, turnstileToken: String) {
        viewModelScope.launch {
            _uiState.value = LoginUiState.Loading
            _uiState.value = when (val result = authRepository.login(sanitizeEmail(email), password, turnstileToken)) {
                is ApiResult.Success -> LoginUiState.LoggedIn
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
