package com.wanli.android.presentation.recipe

import android.net.Uri
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.CommentDto
import com.wanli.android.data.remote.CookingRecordDto
import com.wanli.android.data.remote.CreateCookingRecordRequest
import com.wanli.android.data.repository.PhotoUploadRepository
import com.wanli.android.data.repository.RecipeRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class CommunityViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val recipeRepository: RecipeRepository,
    private val photoUploadRepository: PhotoUploadRepository
) : ViewModel() {
    private val versionId: String = checkNotNull(savedStateHandle["versionId"])
    private val _uiState = MutableStateFlow<CommunityUiState>(CommunityUiState.Loading)
    val uiState: StateFlow<CommunityUiState> = _uiState

    init {
        load()
    }

    fun load() {
        _uiState.value = CommunityUiState.Loading
        viewModelScope.launch {
            val comments = recipeRepository.getComments(versionId)
            val records = recipeRepository.getCookingRecords(versionId)
            _uiState.value = when {
                comments is ApiResult.Success && records is ApiResult.Success ->
                    CommunityUiState.Success(comments.data.items, records.data.items)
                comments is ApiResult.Error -> CommunityUiState.Error(comments.message)
                records is ApiResult.Error -> CommunityUiState.Error(records.message)
                else -> CommunityUiState.Loading
            }
        }
    }

    fun submitComment(content: String) {
        val current = _uiState.value as? CommunityUiState.Success ?: return
        if (content.isBlank() || current.submitting) return
        _uiState.value = current.copy(submitting = true, message = null)
        viewModelScope.launch {
            when (val result = recipeRepository.createComment(versionId, content.trim())) {
                is ApiResult.Success -> load()
                is ApiResult.Error -> finishWithError(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }

    fun deleteComment(commentId: String) {
        mutateThenReload { recipeRepository.deleteComment(commentId) }
    }

    fun submitCookingRecord(
        rating: Int?,
        feedback: String,
        photoUri: Uri?
    ) {
        val current = _uiState.value as? CommunityUiState.Success ?: return
        if (current.submitting) return
        _uiState.value = current.copy(submitting = true, message = null)
        viewModelScope.launch {
            val photoURL = when {
                photoUri == null -> null
                else -> when (val upload = photoUploadRepository.upload(photoUri)) {
                    is ApiResult.Success -> upload.data
                    is ApiResult.Error -> {
                        finishWithError(upload.message)
                        return@launch
                    }
                    is ApiResult.Loading -> return@launch
                }
            }
            val request = CreateCookingRecordRequest(
                rating = rating,
                photo_url = photoURL,
                feedback = feedback.trim().ifBlank { null }
            )
            when (val result = recipeRepository.createCookingRecord(versionId, request)) {
                is ApiResult.Success -> load()
                is ApiResult.Error -> finishWithError(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }

    fun deleteCookingRecord(recordId: String) {
        mutateThenReload { recipeRepository.deleteCookingRecord(recordId) }
    }

    private fun mutateThenReload(call: suspend () -> ApiResult<Unit>) {
        val current = _uiState.value as? CommunityUiState.Success ?: return
        if (current.submitting) return
        _uiState.value = current.copy(submitting = true, message = null)
        viewModelScope.launch {
            when (val result = call()) {
                is ApiResult.Success -> load()
                is ApiResult.Error -> finishWithError(result.message)
                is ApiResult.Loading -> Unit
            }
        }
    }

    private fun finishWithError(message: String) {
        val current = _uiState.value as? CommunityUiState.Success ?: return
        _uiState.value = current.copy(submitting = false, message = message)
    }
}

sealed class CommunityUiState {
    data object Loading : CommunityUiState()
    data class Success(
        val comments: List<CommentDto>,
        val records: List<CookingRecordDto>,
        val submitting: Boolean = false,
        val message: String? = null
    ) : CommunityUiState()
    data class Error(val message: String) : CommunityUiState()
}
