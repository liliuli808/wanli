package com.wanli.android.presentation.recipe

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.ForkVersionRequest
import com.wanli.android.data.remote.SubmitCorrectionRequest
import com.wanli.android.data.repository.RecipeRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

const val EDIT_MODE_CORRECTION = "correction"
const val EDIT_MODE_FORK = "fork"

@HiltViewModel
class RecipeEditViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val recipeRepository: RecipeRepository
) : ViewModel() {
    private val versionId: String = checkNotNull(savedStateHandle["versionId"])
    val mode: String = checkNotNull(savedStateHandle["editMode"])

    private val _uiState = MutableStateFlow<RecipeEditUiState>(RecipeEditUiState.Loading)
    val uiState: StateFlow<RecipeEditUiState> = _uiState

    init {
        load()
    }

    fun load() {
        _uiState.value = RecipeEditUiState.Loading
        viewModelScope.launch {
            when (val result = recipeRepository.getVersionDetail(versionId)) {
                is ApiResult.Success -> _uiState.value = RecipeEditUiState.Editing(result.data.toForm())
                is ApiResult.Error -> _uiState.value = RecipeEditUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }

    fun updateForm(transform: (ProofreadForm) -> ProofreadForm) {
        val state = _uiState.value as? RecipeEditUiState.Editing ?: return
        _uiState.value = state.copy(form = transform(state.form), hint = null)
    }

    fun updateNote(note: String) {
        val state = _uiState.value as? RecipeEditUiState.Editing ?: return
        _uiState.value = state.copy(note = note, hint = null)
    }

    fun addIngredient() = updateForm { it.copy(ingredients = it.ingredients + IngredientForm()) }

    fun removeIngredient(index: Int) = updateForm { form ->
        form.copy(ingredients = form.ingredients.filterIndexed { i, _ -> i != index })
    }

    fun updateIngredient(index: Int, ingredient: IngredientForm) = updateForm { form ->
        form.copy(ingredients = form.ingredients.mapIndexed { i, old -> if (i == index) ingredient else old })
    }

    fun addStep() = updateForm { it.copy(steps = it.steps + StepForm()) }

    fun removeStep(index: Int) = updateForm { form ->
        form.copy(steps = form.steps.filterIndexed { i, _ -> i != index })
    }

    fun updateStep(index: Int, step: StepForm) = updateForm { form ->
        form.copy(steps = form.steps.mapIndexed { i, old -> if (i == index) step else old })
    }

    fun submit() {
        val state = _uiState.value as? RecipeEditUiState.Editing ?: return
        val request = ProofreadViewModel.buildRequest(state.form)
        if (request == null || state.note.isBlank()) {
            _uiState.value = state.copy(
                hint = if (mode == EDIT_MODE_CORRECTION) "请完善菜谱并填写纠错理由" else "请完善菜谱并填写改编说明"
            )
            return
        }

        _uiState.value = state.copy(working = true, hint = null)
        viewModelScope.launch {
            if (mode == EDIT_MODE_CORRECTION) {
                when (val result = recipeRepository.submitCorrection(
                    versionId,
                    SubmitCorrectionRequest(suggested_changes = request, reason = state.note.trim())
                )) {
                    is ApiResult.Success -> _uiState.value = RecipeEditUiState.Submitted(null)
                    is ApiResult.Error -> _uiState.value = state.copy(working = false, hint = result.message)
                    is ApiResult.Loading -> Unit
                }
            } else {
                val forkRequest = ForkVersionRequest(
                    title = request.title,
                    change_log = state.note.trim(),
                    servings = request.servings,
                    estimated_minutes = request.estimated_minutes,
                    instructions = request.instructions,
                    ingredients = request.ingredients,
                    steps = request.steps
                )
                when (val result = recipeRepository.fork(versionId, forkRequest)) {
                    is ApiResult.Success -> _uiState.value = RecipeEditUiState.Submitted(result.data.id)
                    is ApiResult.Error -> _uiState.value = state.copy(working = false, hint = result.message)
                    is ApiResult.Loading -> Unit
                }
            }
        }
    }
}

sealed class RecipeEditUiState {
    data object Loading : RecipeEditUiState()
    data class Editing(
        val form: ProofreadForm,
        val note: String = "",
        val working: Boolean = false,
        val hint: String? = null
    ) : RecipeEditUiState()
    data class Submitted(val versionId: String?) : RecipeEditUiState()
    data class Error(val message: String) : RecipeEditUiState()
}
