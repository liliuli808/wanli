package com.wanli.android.presentation.recipe

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.RecipeDraftDto
import com.wanli.android.data.repository.RecipeRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * 待校对解析草稿列表。
 *
 * 视频解析成功即写库生成未发布的 SOURCE 草稿，草稿不可见、也不会出现在发现页；
 * 这里单独列出，用户可以直接进校对页修改，或原样发布。
 */
@HiltViewModel
class MyDraftsViewModel @Inject constructor(
    private val recipeRepository: RecipeRepository
) : ViewModel() {
    private val _uiState = MutableStateFlow<MyDraftsUiState>(MyDraftsUiState.Loading)
    val uiState: StateFlow<MyDraftsUiState> = _uiState

    init {
        load()
    }

    fun load() {
        _uiState.value = MyDraftsUiState.Loading
        viewModelScope.launch {
            _uiState.value = when (val result = recipeRepository.getMyRecipeDrafts()) {
                is ApiResult.Success -> MyDraftsUiState.Success(result.data.items)
                is ApiResult.Error -> MyDraftsUiState.Error(result.message)
                is ApiResult.Loading -> MyDraftsUiState.Loading
            }
        }
    }

    /**
     * 直接发布：拉取服务端草稿 → 原样提交为发布请求。
     * 草稿内容可能已在其他端校对过，所以这里的 publish 带着完整内容，避免以列表里的
     * 空表单覆盖；发布成功后从列表移除，并发给详情页导航。
     */
    fun publish(draft: RecipeDraftDto) {
        val current = _uiState.value as? MyDraftsUiState.Success ?: return
        val versionId = draft.recipe_version_id
        if (versionId in current.publishingIds) return
        _uiState.value = current.copy(publishingIds = current.publishingIds + versionId, message = null)

        viewModelScope.launch {
            val request = when (val detail = recipeRepository.getVersionDetail(versionId)) {
                is ApiResult.Success -> ProofreadViewModel.buildRequest(detail.data.toForm())
                is ApiResult.Error -> {
                    fail(versionId, detail.message)
                    return@launch
                }
                is ApiResult.Loading -> return@launch
            }
            if (request == null) {
                fail(versionId, "草稿内容不完整，请先进入校对页补充")
                return@launch
            }

            when (val result = recipeRepository.publish(versionId, request)) {
                is ApiResult.Success -> {
                    val latest = _uiState.value as? MyDraftsUiState.Success ?: return@launch
                    _uiState.value = latest.copy(
                        items = latest.items.filterNot { it.recipe_version_id == versionId },
                        publishingIds = latest.publishingIds - versionId,
                        message = "《${draft.title}》已发布",
                        publishedVersionId = result.data.recipe_version_id
                    )
                }
                is ApiResult.Error -> fail(versionId, result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }

    /** 发布跳转已处理后调用，避免配置变化时重复导航。 */
    fun consumePublished() {
        val current = _uiState.value as? MyDraftsUiState.Success ?: return
        if (current.publishedVersionId != null) {
            _uiState.value = current.copy(publishedVersionId = null)
        }
    }

    fun dismissMessage() {
        val current = _uiState.value as? MyDraftsUiState.Success ?: return
        _uiState.value = current.copy(message = null)
    }

    private fun fail(versionId: String, message: String) {
        val latest = _uiState.value as? MyDraftsUiState.Success ?: return
        _uiState.value = latest.copy(
            publishingIds = latest.publishingIds - versionId,
            message = message,
            messageIsError = true
        )
    }
}

sealed class MyDraftsUiState {
    data object Loading : MyDraftsUiState()

    data class Success(
        val items: List<RecipeDraftDto>,
        val publishingIds: Set<String> = emptySet(),
        val message: String? = null,
        val messageIsError: Boolean = false,
        val publishedVersionId: String? = null
    ) : MyDraftsUiState()

    data class Error(val message: String) : MyDraftsUiState()
}
