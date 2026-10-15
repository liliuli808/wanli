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
class AdaptRecipeViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val recipeRepository: RecipeRepository
) : ViewModel() {
    private val versionId: String = checkNotNull(savedStateHandle["versionId"])
    private val _uiState = MutableStateFlow<AdaptRecipeUiState>(AdaptRecipeUiState.Ready)
    val uiState: StateFlow<AdaptRecipeUiState> = _uiState

    fun adapt() {
        if (_uiState.value is AdaptRecipeUiState.Loading) return
        _uiState.value = AdaptRecipeUiState.Loading
        viewModelScope.launch {
            _uiState.value = when (val result = recipeRepository.adapt(versionId)) {
                is ApiResult.Success -> AdaptRecipeUiState.Success(result.data)
                is ApiResult.Error -> AdaptRecipeUiState.Error(
                    code = result.code,
                    message = adaptationErrorMessage(result.code, result.message)
                )
                is ApiResult.Loading -> AdaptRecipeUiState.Loading
            }
        }
    }
}

sealed class AdaptRecipeUiState {
    data object Ready : AdaptRecipeUiState()
    data object Loading : AdaptRecipeUiState()
    data class Success(val detail: RecipeVersionData) : AdaptRecipeUiState()
    data class Error(val code: String, val message: String) : AdaptRecipeUiState()
}

internal fun adaptationErrorMessage(code: String, fallback: String): String = when (code) {
    "ALLERGY_BLOCKED" -> "由于过敏源安全限制，这份菜谱无法适配。建议放弃制作，不要自行尝试替换。"
    "DIETARY_BLOCKED" -> "当前忌口或医疗饮食限制无法被安全满足，未生成适配版。"
    "KITCHEN_FIT_CANNOT_FORK" -> "个人适配版是终端版本，不能再次派生适配。"
    else -> fallback
}
