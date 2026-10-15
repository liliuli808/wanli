package com.wanli.android.presentation.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.MyCookingRecordDto
import com.wanli.android.data.repository.AuthRepository
import com.wanli.android.data.repository.SocialRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

sealed interface ProfileUiState {
    data object Loading : ProfileUiState
    data class Error(val message: String) : ProfileUiState
    data class Success(
        val nickname: String,
        val email: String,
        val avatarUrl: String?,
        val cookingRecords: List<MyCookingRecordDto>,
    ) : ProfileUiState
}

@HiltViewModel
class ProfileViewModel @Inject constructor(
    private val authRepository: AuthRepository,
    private val socialRepository: SocialRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow<ProfileUiState>(ProfileUiState.Loading)
    val uiState: StateFlow<ProfileUiState> = _uiState.asStateFlow()

    init {
        load()
    }

    fun load() {
        viewModelScope.launch {
            _uiState.value = ProfileUiState.Loading
            when (val me = authRepository.getMe()) {
                is ApiResult.Success -> {
                    val records = (socialRepository.getMyCookingRecords() as? ApiResult.Success)?.data
                        ?: emptyList()
                    _uiState.value = ProfileUiState.Success(
                        nickname = me.data.nickname,
                        email = me.data.email,
                        avatarUrl = me.data.avatar_url,
                        cookingRecords = records,
                    )
                }
                is ApiResult.Error -> _uiState.value = ProfileUiState.Error(me.message)
                is ApiResult.Loading -> Unit
            }
        }
    }

    fun updateNickname(nickname: String) {
        val current = _uiState.value as? ProfileUiState.Success ?: return
        viewModelScope.launch {
            when (val result = authRepository.updateProfile(nickname, current.avatarUrl)) {
                is ApiResult.Success ->
                    (_uiState.value as? ProfileUiState.Success)?.let {
                        _uiState.value = it.copy(nickname = result.data.nickname)
                    }
                else -> Unit // 失败提示后续以 snackbar 补充，本期静默保持旧值
            }
        }
    }
}
