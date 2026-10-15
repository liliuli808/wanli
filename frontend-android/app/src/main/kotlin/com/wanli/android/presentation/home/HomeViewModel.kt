package com.wanli.android.presentation.home

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.CookingFeedItemDto
import com.wanli.android.data.repository.SocialRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

sealed interface HomeFeedUiState {
    data object Loading : HomeFeedUiState
    data class Error(val message: String) : HomeFeedUiState
    data class Success(val items: List<CookingFeedItemDto>) : HomeFeedUiState
}

@HiltViewModel
class HomeViewModel @Inject constructor(
    private val socialRepository: SocialRepository,
) : ViewModel() {

    private val _feedState = MutableStateFlow<HomeFeedUiState>(HomeFeedUiState.Loading)
    val feedState: StateFlow<HomeFeedUiState> = _feedState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _feedState.value = HomeFeedUiState.Loading
            when (val result = socialRepository.getCookingFeed()) {
                is ApiResult.Success -> _feedState.value = HomeFeedUiState.Success(result.data)
                is ApiResult.Error -> _feedState.value = HomeFeedUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }
}
