package com.wanli.android.presentation.recipe

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.PendingCorrectionDto
import com.wanli.android.data.remote.ReviewCorrectionRequest
import com.wanli.android.data.repository.RecipeRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class PendingCorrectionsViewModel @Inject constructor(
    private val recipeRepository: RecipeRepository
) : ViewModel() {
    private val _uiState = MutableStateFlow<PendingCorrectionsUiState>(PendingCorrectionsUiState.Loading)
    val uiState: StateFlow<PendingCorrectionsUiState> = _uiState

    init {
        load()
    }

    fun load() {
        _uiState.value = PendingCorrectionsUiState.Loading
        viewModelScope.launch {
            when (val result = recipeRepository.getPendingCorrections()) {
                is ApiResult.Success -> _uiState.value = PendingCorrectionsUiState.Success(result.data.items)
                is ApiResult.Error -> _uiState.value = PendingCorrectionsUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }

    fun review(proposalId: String, approve: Boolean, rejectionReason: String = "") {
        val state = _uiState.value as? PendingCorrectionsUiState.Success ?: return
        if (!approve && rejectionReason.isBlank()) {
            _uiState.value = state.copy(message = "拒绝时请填写理由")
            return
        }
        _uiState.value = state.copy(workingId = proposalId, message = null)
        viewModelScope.launch {
            val request = ReviewCorrectionRequest(
                action = if (approve) "approve" else "reject",
                rejection_reason = rejectionReason.trim()
            )
            when (val result = recipeRepository.reviewCorrection(proposalId, request)) {
                is ApiResult.Success -> _uiState.value = state.copy(
                    items = state.items.filterNot { it.id == proposalId },
                    message = if (approve) "已通过并生成新的来源版本" else "已拒绝"
                )
                is ApiResult.Error -> _uiState.value = state.copy(message = result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }
}

sealed class PendingCorrectionsUiState {
    data object Loading : PendingCorrectionsUiState()
    data class Success(
        val items: List<PendingCorrectionDto>,
        val workingId: String? = null,
        val message: String? = null
    ) : PendingCorrectionsUiState()
    data class Error(val message: String) : PendingCorrectionsUiState()
}
