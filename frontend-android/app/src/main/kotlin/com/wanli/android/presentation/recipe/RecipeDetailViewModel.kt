package com.wanli.android.presentation.recipe

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.RecipeVersionData
import com.wanli.android.data.repository.RecipeRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class RecipeDetailViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val recipeRepository: RecipeRepository
) : ViewModel() {

    private val versionId: String = checkNotNull(savedStateHandle["versionId"])

    private val _uiState = MutableStateFlow<RecipeDetailUiState>(RecipeDetailUiState.Loading)
    val uiState: StateFlow<RecipeDetailUiState> = _uiState

    init {
        load()
    }

    fun load() {
        _uiState.value = RecipeDetailUiState.Loading
        viewModelScope.launch {
            when (val result = recipeRepository.getVersionDetail(versionId)) {
                is ApiResult.Success -> _uiState.value = RecipeDetailUiState.Success(result.data)
                is ApiResult.Error -> _uiState.value = RecipeDetailUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }
}

sealed class RecipeDetailUiState {
    data object Loading : RecipeDetailUiState()
    data class Success(val detail: RecipeVersionData) : RecipeDetailUiState()
    data class Error(val message: String) : RecipeDetailUiState()
}
