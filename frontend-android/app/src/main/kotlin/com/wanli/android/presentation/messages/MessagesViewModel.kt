package com.wanli.android.presentation.messages

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.MessageDto
import com.wanli.android.data.repository.SocialRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

sealed interface MessagesUiState {
    data object Loading : MessagesUiState
    data class Error(val message: String) : MessagesUiState
    data class Success(val items: List<MessageDto>) : MessagesUiState
}

@HiltViewModel
class MessagesViewModel @Inject constructor(
    private val socialRepository: SocialRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow<MessagesUiState>(MessagesUiState.Loading)
    val uiState: StateFlow<MessagesUiState> = _uiState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.value = MessagesUiState.Loading
            when (val result = socialRepository.getMessages()) {
                is ApiResult.Success -> _uiState.value = MessagesUiState.Success(result.data)
                is ApiResult.Error -> _uiState.value = MessagesUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }
}
