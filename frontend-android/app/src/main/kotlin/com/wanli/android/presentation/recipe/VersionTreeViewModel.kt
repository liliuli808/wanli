package com.wanli.android.presentation.recipe

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.VersionTreeNodeDto
import com.wanli.android.data.repository.RecipeRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class VersionTreeViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val recipeRepository: RecipeRepository
) : ViewModel() {
    private val recipeId: String = checkNotNull(savedStateHandle["recipeId"])
    private val _uiState = MutableStateFlow<VersionTreeUiState>(VersionTreeUiState.Loading)
    val uiState: StateFlow<VersionTreeUiState> = _uiState

    init {
        load()
    }

    fun load() {
        _uiState.value = VersionTreeUiState.Loading
        viewModelScope.launch {
            when (val result = recipeRepository.getVersionTree(recipeId)) {
                is ApiResult.Success -> _uiState.value = VersionTreeUiState.Success(result.data.nodes)
                is ApiResult.Error -> _uiState.value = VersionTreeUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }
}

sealed class VersionTreeUiState {
    data object Loading : VersionTreeUiState()
    data class Success(val nodes: List<VersionTreeNodeDto>) : VersionTreeUiState()
    data class Error(val message: String) : VersionTreeUiState()
}
