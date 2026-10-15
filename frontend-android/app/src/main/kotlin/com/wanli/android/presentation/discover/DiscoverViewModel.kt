package com.wanli.android.presentation.discover

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.RecipeSummaryDto
import com.wanli.android.data.repository.RecipeRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class DiscoverViewModel @Inject constructor(
    private val recipeRepository: RecipeRepository
) : ViewModel() {

    private val _uiState = MutableStateFlow<DiscoverUiState>(DiscoverUiState.Loading)
    val uiState: StateFlow<DiscoverUiState> = _uiState

    var currentSort: String = SORT_LATEST
        private set

    init {
        load(SORT_LATEST)
    }

    fun setSort(sort: String) {
        if (sort == currentSort && _uiState.value is DiscoverUiState.Success) return
        load(sort)
    }

    fun refresh() = load(currentSort)

    private fun load(sort: String) {
        currentSort = sort
        _uiState.value = DiscoverUiState.Loading
        viewModelScope.launch {
            when (val result = recipeRepository.listRecipes(sort, page = 1, pageSize = PAGE_SIZE)) {
                is ApiResult.Success -> _uiState.value = DiscoverUiState.Success(result.data.items, sort)
                is ApiResult.Error -> _uiState.value = DiscoverUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }

    companion object {
        const val SORT_LATEST = "latest"
        const val SORT_HOTTEST = "hottest"
        const val PAGE_SIZE = 20
    }
}

sealed class DiscoverUiState {
    data object Loading : DiscoverUiState()
    data class Success(val items: List<RecipeSummaryDto>, val sort: String) : DiscoverUiState()
    data class Error(val message: String) : DiscoverUiState()
}
