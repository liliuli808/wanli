package com.wanli.android.presentation.recipe

import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.IngredientDto
import com.wanli.android.data.remote.RecipeVersionData
import com.wanli.android.data.remote.StepDto
import com.wanli.android.data.remote.UpdateDraftRequest
import com.wanli.android.data.repository.RecipeRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/** 校对表单中的食材行（数值字段以字符串承载，便于直接绑定输入框）。 */
data class IngredientForm(
    val name: String = "",
    val amount: String = "",
    val category: String = CATEGORY_MAIN,
    val isExplicit: Boolean = true
)

/** 校对表单中的步骤行。 */
data class StepForm(
    val description: String = "",
    val videoTimestamp: String = "",
    val heat: String = ""
)

data class ProofreadForm(
    val title: String = "",
    val servings: String = "",
    val estimatedMinutes: String = "",
    val instructions: String = "",
    val ingredients: List<IngredientForm> = listOf(IngredientForm()),
    val steps: List<StepForm> = listOf(StepForm())
)

const val CATEGORY_MAIN = "MAIN"
const val CATEGORY_AUXILIARY = "AUXILIARY"
const val CATEGORY_SEASONING = "SEASONING"

val CATEGORY_ORDER = listOf(CATEGORY_MAIN, CATEGORY_AUXILIARY, CATEGORY_SEASONING)

fun categoryLabel(category: String): String = when (category) {
    CATEGORY_MAIN -> "主料"
    CATEGORY_AUXILIARY -> "辅料"
    CATEGORY_SEASONING -> "调味品"
    else -> category
}

@HiltViewModel
class ProofreadViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val recipeRepository: RecipeRepository
) : ViewModel() {

    private val versionId: String = checkNotNull(savedStateHandle["versionId"])

    private val _uiState = MutableStateFlow<ProofreadUiState>(ProofreadUiState.Loading)
    val uiState: StateFlow<ProofreadUiState> = _uiState

    /** 发布前的编辑态快照，供 consumePublished 恢复表单。 */
    private var lastEditingState: ProofreadUiState.Editing? = null

    init {
        load()
    }

    fun load() {
        _uiState.value = ProofreadUiState.Loading
        viewModelScope.launch {
            when (val result = recipeRepository.getVersionDetail(versionId)) {
                is ApiResult.Success -> _uiState.value = ProofreadUiState.Editing(
                    form = result.data.toForm(),
                    coverUrl = result.data.cover_url,
                    upNickname = result.data.up_nickname
                )
                is ApiResult.Error -> _uiState.value = ProofreadUiState.Error(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }

    fun updateForm(transform: (ProofreadForm) -> ProofreadForm) {
        val state = _uiState.value as? ProofreadUiState.Editing ?: return
        _uiState.value = state.copy(form = transform(state.form), hint = null, isError = false)
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

    /** 保存草稿（PUT /draft），成功后停留在编辑态并提示。 */
    fun save() {
        val state = _uiState.value as? ProofreadUiState.Editing ?: return
        val request = buildRequest(state.form) ?: run {
            _uiState.value = state.copy(hint = "请填写菜名，并至少保留一条食材和一个步骤", isError = true)
            return
        }
        _uiState.value = state.copy(working = true)
        viewModelScope.launch {
            when (val result = recipeRepository.updateDraft(versionId, request)) {
                is ApiResult.Success -> {
                    // 保留请求期间的本地输入，仅回填服务端字段与提示
                    val current = _uiState.value as? ProofreadUiState.Editing ?: return@launch
                    _uiState.value = current.copy(
                        working = false,
                        hint = "草稿已保存",
                        isError = false,
                        coverUrl = result.data.cover_url,
                        upNickname = result.data.up_nickname
                    )
                }
                is ApiResult.Error -> {
                    // 基于当前状态回写，避免覆盖请求期间的编辑
                    val current = _uiState.value as? ProofreadUiState.Editing ?: return@launch
                    _uiState.value = current.copy(working = false, hint = result.message, isError = true)
                }
                is ApiResult.Loading -> Unit
            }
        }
    }

    /** 原子提交当前校对内容并发布，避免“保存成功但发布失败”的中间状态。 */
    fun publish() {
        val state = _uiState.value as? ProofreadUiState.Editing ?: return
        val request = buildRequest(state.form) ?: run {
            _uiState.value = state.copy(hint = "请填写菜名，并至少保留一条食材和一个步骤", isError = true)
            return
        }
        lastEditingState = state
        _uiState.value = state.copy(working = true)
        viewModelScope.launch {
            when (val result = recipeRepository.publish(versionId, request)) {
                is ApiResult.Success -> _uiState.value = ProofreadUiState.Published(versionId)
                is ApiResult.Error -> {
                    val current = _uiState.value as? ProofreadUiState.Editing ?: return@launch
                    _uiState.value = current.copy(working = false, hint = result.message, isError = true)
                }
                is ApiResult.Loading -> Unit
            }
        }
    }

    /** 发布跳转已处理后调用：回到编辑态，避免配置变化时重复触发导航。 */
    fun consumePublished() {
        if (_uiState.value !is ProofreadUiState.Published) return
        val editing = lastEditingState ?: return
        _uiState.value = editing.copy(working = false)
    }

    companion object {
        /** 将表单组装为草稿更新请求；校验不通过返回 null。 */
        fun buildRequest(form: ProofreadForm): UpdateDraftRequest? {
            if (form.title.isBlank()) return null
            val ingredients = form.ingredients
                .filter { it.name.isNotBlank() }
                .map {
                    IngredientDto(
                        name = it.name.trim(),
                        amount = it.amount.trim().ifEmpty { null },
                        category = it.category,
                        is_explicit = it.isExplicit
                    )
                }
            val steps = form.steps
                .filter { it.description.isNotBlank() }
                .mapIndexed { index, step ->
                    StepDto(
                        step_order = index + 1,
                        description = step.description.trim(),
                        video_timestamp = step.videoTimestamp.trim().toIntOrNull(),
                        duration_seconds = null,
                        heat = step.heat.trim().ifEmpty { null }
                    )
                }
            if (ingredients.isEmpty() || steps.isEmpty()) return null
            return UpdateDraftRequest(
                title = form.title.trim(),
                servings = form.servings.trim().ifEmpty { null },
                estimated_minutes = form.estimatedMinutes.trim().toIntOrNull(),
                instructions = form.instructions.trim().ifEmpty { null },
                ingredients = ingredients,
                steps = steps
            )
        }
    }
}

internal fun RecipeVersionData.toForm(): ProofreadForm = ProofreadForm(
    title = title,
    servings = servings.orEmpty(),
    estimatedMinutes = estimated_minutes?.toString().orEmpty(),
    instructions = instructions.orEmpty(),
    ingredients = ingredients.map {
        IngredientForm(
            name = it.name,
            amount = it.amount.orEmpty(),
            category = it.category,
            isExplicit = it.is_explicit
        )
    }.ifEmpty { listOf(IngredientForm()) },
    steps = steps.map {
        StepForm(
            description = it.description,
            videoTimestamp = it.video_timestamp?.toString().orEmpty(),
            heat = it.heat.orEmpty()
        )
    }.ifEmpty { listOf(StepForm()) }
)

sealed class ProofreadUiState {
    data object Loading : ProofreadUiState()
    data class Editing(
        val form: ProofreadForm,
        val working: Boolean = false,
        val hint: String? = null,
        val isError: Boolean = false,
        val coverUrl: String? = null,
        val upNickname: String = ""
    ) : ProofreadUiState()
    data class Published(val versionId: String) : ProofreadUiState()
    data class Error(val message: String) : ProofreadUiState()
}
