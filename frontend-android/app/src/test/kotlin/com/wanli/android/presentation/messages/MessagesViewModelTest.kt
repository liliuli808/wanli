package com.wanli.android.presentation.messages

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.MessageDto
import com.wanli.android.data.repository.SocialRepository
import io.mockk.coEvery
import io.mockk.mockk
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.ExperimentalCoroutinesApi
import kotlinx.coroutines.test.UnconfinedTestDispatcher
import kotlinx.coroutines.test.resetMain
import kotlinx.coroutines.test.runTest
import kotlinx.coroutines.test.setMain
import org.junit.After
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

@OptIn(ExperimentalCoroutinesApi::class)
class MessagesViewModelTest {

    private val testDispatcher = UnconfinedTestDispatcher()
    private lateinit var socialRepository: SocialRepository

    private fun message(id: String, type: String) = MessageDto(
        type = type,
        id = id,
        actorNickname = "小厨",
        text = "很好吃",
        rating = null,
        recipeVersionId = "v1",
        recipeTitle = "红烧肉",
        createdAt = "2026-07-01T10:00:00Z",
    )

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        socialRepository = mockk()
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `load populates message list`() = runTest {
        coEvery { socialRepository.getMessages() } returns
            ApiResult.Success(listOf(message("1", "comment"), message("2", "cooking_record")))

        val viewModel = MessagesViewModel(socialRepository)

        val state = viewModel.uiState.value
        assertTrue(state is MessagesUiState.Success)
        assertEquals(2, (state as MessagesUiState.Success).items.size)
    }

    @Test
    fun `load shows error when getMessages fails`() = runTest {
        coEvery { socialRepository.getMessages() } returns
            ApiResult.Error("NETWORK_ERROR", "网络错误")

        val viewModel = MessagesViewModel(socialRepository)

        val state = viewModel.uiState.value
        assertTrue(state is MessagesUiState.Error)
        assertEquals("网络错误", (state as MessagesUiState.Error).message)
    }

    @Test
    fun `refresh reloads messages`() = runTest {
        coEvery { socialRepository.getMessages() } returnsMany listOf(
            ApiResult.Success(listOf(message("1", "comment"))),
            ApiResult.Success(listOf(message("1", "comment"), message("2", "correction_pending"))),
        )

        val viewModel = MessagesViewModel(socialRepository)
        assertEquals(1, (viewModel.uiState.value as MessagesUiState.Success).items.size)

        viewModel.refresh()

        assertEquals(2, (viewModel.uiState.value as MessagesUiState.Success).items.size)
    }
}
