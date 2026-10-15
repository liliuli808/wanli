package com.wanli.android.presentation.profile

import com.wanli.android.data.remote.ApiResult
import com.wanli.android.data.remote.MyCookingRecordDto
import com.wanli.android.data.remote.UserDto
import com.wanli.android.data.repository.AuthRepository
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
class ProfileViewModelTest {

    private val testDispatcher = UnconfinedTestDispatcher()
    private lateinit var authRepository: AuthRepository
    private lateinit var socialRepository: SocialRepository

    private fun user(nickname: String) = UserDto(
        id = "u1", email = "a@b.c", nickname = nickname, avatar_url = null,
    )

    private fun record(id: String, photoUrl: String?) = MyCookingRecordDto(
        id = id, rating = 5, photoUrl = photoUrl, feedback = null, createdAt = null,
        recipeVersionId = "v1", recipeTitle = "红烧肉", coverUrl = null,
    )

    @Before
    fun setup() {
        Dispatchers.setMain(testDispatcher)
        authRepository = mockk()
        socialRepository = mockk()
    }

    @After
    fun tearDown() {
        Dispatchers.resetMain()
    }

    @Test
    fun `load populates identity and photo wall`() = runTest {
        coEvery { authRepository.getMe() } returns ApiResult.Success(user("小厨"))
        coEvery { socialRepository.getMyCookingRecords(any()) } returns
            ApiResult.Success(listOf(record("1", "https://img/1.jpg")))

        val viewModel = ProfileViewModel(authRepository, socialRepository)

        val state = viewModel.uiState.value
        assertTrue(state is ProfileUiState.Success)
        state as ProfileUiState.Success
        assertEquals("小厨", state.nickname)
        assertEquals("a@b.c", state.email)
        assertEquals(1, state.cookingRecords.size)
    }

    @Test
    fun `load shows error when getMe fails`() = runTest {
        coEvery { authRepository.getMe() } returns ApiResult.Error("NETWORK_ERROR", "网络错误")

        val viewModel = ProfileViewModel(authRepository, socialRepository)

        val state = viewModel.uiState.value
        assertTrue(state is ProfileUiState.Error)
        assertEquals("网络错误", (state as ProfileUiState.Error).message)
    }

    @Test
    fun `load succeeds with empty photo wall when records fetch fails`() = runTest {
        coEvery { authRepository.getMe() } returns ApiResult.Success(user("小厨"))
        coEvery { socialRepository.getMyCookingRecords(any()) } returns
            ApiResult.Error("NETWORK_ERROR", "网络错误")

        val viewModel = ProfileViewModel(authRepository, socialRepository)

        val state = viewModel.uiState.value
        assertTrue(state is ProfileUiState.Success)
        state as ProfileUiState.Success
        assertEquals("小厨", state.nickname)
        assertTrue(state.cookingRecords.isEmpty())
    }

    @Test
    fun `updateNickname refreshes identity`() = runTest {
        coEvery { authRepository.getMe() } returns ApiResult.Success(user("旧"))
        coEvery { socialRepository.getMyCookingRecords(any()) } returns ApiResult.Success(emptyList())
        coEvery { authRepository.updateProfile("新昵称", null) } returns ApiResult.Success(user("新昵称"))

        val viewModel = ProfileViewModel(authRepository, socialRepository)
        viewModel.updateNickname("新昵称")

        assertEquals("新昵称", (viewModel.uiState.value as ProfileUiState.Success).nickname)
    }
}
