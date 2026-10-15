package com.wanli.android.presentation.kitchen

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.RestrictionDto
import com.wanli.android.data.repository.KitchenRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class DietaryViewModel @Inject constructor(
    private val kitchenRepository: KitchenRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow<DietaryUiState>(DietaryUiState.Loading)
    val uiState: StateFlow<DietaryUiState> = _uiState

    init {
        loadRestrictions()
    }

    fun loadRestrictions() {
        viewModelScope.launch {
            _uiState.value = DietaryUiState.Loading
            _uiState.value = when (val result = kitchenRepository.getRestrictions()) {
                is ApiResult.Success -> DietaryUiState.Success(result.data)
                is ApiResult.Error -> DietaryUiState.Error(result.message)
                is ApiResult.Loading -> DietaryUiState.Loading
            }
        }
    }

    fun saveRestrictions(restrictions: List<RestrictionDto>) {
        viewModelScope.launch {
            _uiState.value = DietaryUiState.Loading
            _uiState.value = when (val result = kitchenRepository.updateRestrictions(restrictions)) {
                is ApiResult.Success -> DietaryUiState.Success(result.data)
                is ApiResult.Error -> DietaryUiState.Error(result.message)
                is ApiResult.Loading -> DietaryUiState.Loading
            }
        }
    }
}

sealed class DietaryUiState {
    data object Loading : DietaryUiState()
    data class Success(val restrictions: List<RestrictionDto>) : DietaryUiState()
    data class Error(val message: String) : DietaryUiState()
}
