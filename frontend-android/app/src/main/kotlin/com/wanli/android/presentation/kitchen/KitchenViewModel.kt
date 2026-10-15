package com.wanli.android.presentation.kitchen

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.KitchenData
import com.wanli.android.data.repository.KitchenRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class KitchenViewModel @Inject constructor(
    private val kitchenRepository: KitchenRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow<KitchenUiState>(KitchenUiState.Loading)
    val uiState: StateFlow<KitchenUiState> = _uiState

    init {
        loadKitchen()
    }

    fun loadKitchen() {
        viewModelScope.launch {
            _uiState.value = KitchenUiState.Loading
            _uiState.value = when (val result = kitchenRepository.getKitchen()) {
                is ApiResult.Success -> KitchenUiState.Success(
                    ingredients = result.data.owned_ingredients,
                    tools = result.data.owned_tools
                )
                is ApiResult.Error -> KitchenUiState.Error(result.message)
                is ApiResult.Loading -> KitchenUiState.Loading
            }
        }
    }

    fun saveKitchen(ingredients: List<String>, tools: List<String>) {
        viewModelScope.launch {
            _uiState.value = KitchenUiState.Loading
            _uiState.value = when (val result = kitchenRepository.updateKitchen(ingredients, tools)) {
                is ApiResult.Success -> KitchenUiState.Success(
                    ingredients = result.data.owned_ingredients,
                    tools = result.data.owned_tools
                )
                is ApiResult.Error -> KitchenUiState.Error(result.message)
                is ApiResult.Loading -> KitchenUiState.Loading
            }
        }
    }
}

sealed class KitchenUiState {
    data object Loading : KitchenUiState()
    data class Success(
        val ingredients: List<String>,
        val tools: List<String>
    ) : KitchenUiState()
    data class Error(val message: String) : KitchenUiState()
}
