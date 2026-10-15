package com.wanli.android.presentation.recipe

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.MyAdaptationDto
import com.wanli.android.data.repository.RecipeRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class MyAdaptationsViewModel @Inject constructor(
    private val recipeRepository: RecipeRepository
) : ViewModel() {
    private val _uiState = MutableStateFlow<MyAdaptationsUiState>(MyAdaptationsUiState.Loading)
    val uiState: StateFlow<MyAdaptationsUiState> = _uiState

    init {
        load()
    }

    fun load() {
        _uiState.value = MyAdaptationsUiState.Loading
        viewModelScope.launch {
            _uiState.value = when (val result = recipeRepository.getMyAdaptations()) {
                is ApiResult.Success -> MyAdaptationsUiState.Success(result.data.items)
                is ApiResult.Error -> MyAdaptationsUiState.Error(result.message)
                is ApiResult.Loading -> MyAdaptationsUiState.Loading
            }
        }
    }

    fun setVisibility(versionId: String, isPublic: Boolean) {
        val current = _uiState.value as? MyAdaptationsUiState.Success ?: return
        if (versionId in current.updatingIds) return
        _uiState.value = current.copy(
            updatingIds = current.updatingIds + versionId,
            message = null
        )
        viewModelScope.launch {
            when (val result = recipeRepository.setVisibility(versionId, isPublic)) {
                is ApiResult.Success -> {
                    val latest = _uiState.value as? MyAdaptationsUiState.Success ?: return@launch
                    _uiState.value = latest.copy(
                        items = latest.items.map {
                            if (it.id == versionId) it.copy(is_public = result.data.is_public) else it
                        },
                        updatingIds = latest.updatingIds - versionId
                    )
                }
                is ApiResult.Error -> {
                    val latest = _uiState.value as? MyAdaptationsUiState.Success ?: return@launch
                    _uiState.value = latest.copy(
                        updatingIds = latest.updatingIds - versionId,
                        message = result.message
                    )
                }
                is ApiResult.Loading -> Unit
            }
        }
    }
}

sealed class MyAdaptationsUiState {
    data object Loading : MyAdaptationsUiState()
    data class Success(
        val items: List<MyAdaptationDto>,
        val updatingIds: Set<String> = emptySet(),
        val message: String? = null
    ) : MyAdaptationsUiState()
    data class Error(val message: String) : MyAdaptationsUiState()
}
