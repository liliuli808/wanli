package com.wanli.android.presentation

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.wanli.android.data.local.TokenDataStore
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class MainViewModel @Inject constructor(
    private val tokenDataStore: TokenDataStore
) : ViewModel() {

    val isLoggedIn: Flow<Boolean> = tokenDataStore.isLoggedIn

    fun logout() {
        viewModelScope.launch {
            tokenDataStore.clearToken()
        }
    }
}
