package com.wanli.android.presentation.recipe

import android.net.Uri
import androidx.lifecycle.SavedStateHandle
import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.CommentDto
import com.wanli.android.data.remote.CommentsData
import com.wanli.android.data.remote.CookingRecordDto
import com.wanli.android.data.remote.CookingRecordsData
import com.wanli.android.data.remote.CreateCookingRecordRequest
import com.wanli.android.data.repository.PhotoUploadRepository
import com.wanli.android.data.repository.RecipeRepository
import io.mockk.coEvery
import io.mockk.coVerify
import io.mockk.mockk
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.StandardTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class CommunityViewModelTest {
    private val dispatcher = StandardTestDispatcher()
    private lateinit var recipeRepository: RecipeRepository
    private lateinit var photoUploadRepository: PhotoUploadRepository

    @Before
    fun setUp() {
        Dispatchers.setMain(dispatcher)
        recipeRepository = mockk()
        photoUploadRepository = mockk()
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    private fun stubLists(
        comments: List<CommentDto> = emptyList(),
        records: List<CookingRecordDto> = emptyList()
    ) {
        coEvery { recipeRepository.getComments("version-1") } returns
            ApiResult.Success(CommentsData(comments))
        coEvery { recipeRepository.getCookingRecords("version-1") } returns
            ApiResult.Success(CookingRecordsData(records))
    }

    private fun viewModel() = CommunityViewModel(
        SavedStateHandle(mapOf("versionId" to "version-1")),
        recipeRepository,
        photoUploadRepository
    )

    @Test
    fun `loads public comments and cooking records`() = runTest {
        stubLists(
            comments = listOf(CommentDto("comment-1", "很好吃", "小碗", true)),
            records = listOf(CookingRecordDto("record-1", rating = 5, is_mine = false))
        )

        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value as CommunityUiState.Success
        assertEquals("很好吃", state.comments.single().content)
        assertEquals(5, state.records.single().rating)
    }

    @Test
    fun `submits trimmed comment and reloads`() = runTest {
        stubLists()
        coEvery { recipeRepository.createComment("version-1", "很好吃") } returns
            ApiResult.Success(CommentDto("comment-1", "很好吃", is_mine = true))
        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.submitComment("  很好吃  ")
        testScheduler.advanceUntilIdle()

        coVerify(exactly = 1) { recipeRepository.createComment("version-1", "很好吃") }
        assertTrue(vm.uiState.value is CommunityUiState.Success)
    }

    @Test
    fun `uploads selected photo before creating cooking record`() = runTest {
        stubLists()
        val uri = mockk<Uri>()
        val publicURL = "http://10.0.2.2:9000/wanli-assets/uploads/records/photo.jpg"
        coEvery { photoUploadRepository.upload(uri) } returns ApiResult.Success(publicURL)
        coEvery {
            recipeRepository.createCookingRecord(
                "version-1",
                match<CreateCookingRecordRequest> {
                    it.rating == 5 && it.photo_url == publicURL && it.feedback == "成功"
                }
            )
        } returns ApiResult.Success(CookingRecordDto("record-1", rating = 5))
        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.submitCookingRecord(5, "成功", uri)
        testScheduler.advanceUntilIdle()

        coVerify(exactly = 1) { photoUploadRepository.upload(uri) }
        assertFalse((vm.uiState.value as CommunityUiState.Success).submitting)
    }

    @Test
    fun `delete failure preserves content and shows message`() = runTest {
        val comment = CommentDto("comment-1", "我的评论", is_mine = true)
        stubLists(comments = listOf(comment))
        coEvery { recipeRepository.deleteComment("comment-1") } returns
            ApiResult.Error("FORBIDDEN", "无权操作")
        val vm = viewModel()
        testScheduler.advanceUntilIdle()

        vm.deleteComment("comment-1")
        testScheduler.advanceUntilIdle()

        val state = vm.uiState.value as CommunityUiState.Success
        assertEquals("我的评论", state.comments.single().content)
        assertEquals("无权操作", state.message)
    }
}
