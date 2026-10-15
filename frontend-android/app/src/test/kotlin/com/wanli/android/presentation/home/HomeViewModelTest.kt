package com.wanli.android.presentation.home

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.CookingFeedItemDto
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
class HomeViewModelTest {

    private val testDispatcher = UnconfinedTestDispatcher()
    private lateinit var socialRepository: SocialRepository

    private fun feedItem(id: String) = CookingFeedItemDto(
        id = id, rating = 5, photoUrl = null, feedback = "好吃",
        authorNickname = "小厨", createdAt = null,
        recipeVersionId = "v1", recipeId = "r1", recipeTitle = "红烧肉", coverUrl = null,
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
    fun `init loads feed successfully`() = runTest {
        coEvery { socialRepository.getCookingFeed(any()) } returns ApiResult.Success(listOf(feedItem("1")))

        val viewModel = HomeViewModel(socialRepository)

        val state = viewModel.feedState.value
        assertTrue(state is HomeFeedUiState.Success)
        assertEquals(1, (state as HomeFeedUiState.Success).items.size)
    }

    @Test
    fun `init shows error when feed fails`() = runTest {
        coEvery { socialRepository.getCookingFeed(any()) } returns ApiResult.Error("NETWORK_ERROR", "网络错误")

        val viewModel = HomeViewModel(socialRepository)

        val state = viewModel.feedState.value
        assertTrue(state is HomeFeedUiState.Error)
        assertEquals("网络错误", (state as HomeFeedUiState.Error).message)
    }

    @Test
    fun `refresh recovers after error`() = runTest {
        coEvery { socialRepository.getCookingFeed(any()) } returns ApiResult.Error("NETWORK_ERROR", "网络错误")
        val viewModel = HomeViewModel(socialRepository)
        assertTrue(viewModel.feedState.value is HomeFeedUiState.Error)

        coEvery { socialRepository.getCookingFeed(any()) } returns ApiResult.Success(listOf(feedItem("2")))
        viewModel.refresh()

        val state = viewModel.feedState.value
        assertTrue(state is HomeFeedUiState.Success)
        assertEquals(1, (state as HomeFeedUiState.Success).items.size)
    }
}
