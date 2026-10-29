package com.wanli.android.data.local

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map

private val Context.dataStore: DataStore<Preferences> by preferencesDataStore(name = "wanli_prefs")

class TokenDataStore(private val context: Context) {

    companion object {
        val TOKEN_KEY = stringPreferencesKey("auth_token")
        val USER_ID_KEY = stringPreferencesKey("user_id")
        val EMAIL_KEY = stringPreferencesKey("email")

        /** 「记住密码」用的键，与登录态的 EMAIL_KEY 分开存，退出登录后仍然保留。 */
        val SAVED_EMAIL_KEY = stringPreferencesKey("saved_email")
        val SAVED_PASSWORD_KEY = stringPreferencesKey("saved_password_encrypted")
    }

    val token: Flow<String?> = context.dataStore.data.map { prefs ->
        prefs[TOKEN_KEY]
    }

    val email: Flow<String?> = context.dataStore.data.map { prefs ->
        prefs[EMAIL_KEY]
    }

    val isLoggedIn: Flow<Boolean> = token.map { it != null }

    /** 上次勾选「记住密码」时保存的账号密码；解不出来（换机、清数据）视为没有。 */
    val savedCredentials: Flow<SavedCredentials?> = context.dataStore.data.map { prefs ->
        val email = prefs[SAVED_EMAIL_KEY]
        val encrypted = prefs[SAVED_PASSWORD_KEY]
        if (email.isNullOrBlank() || encrypted.isNullOrBlank()) {
            null
        } else {
            CredentialCipher.decrypt(encrypted)?.let { SavedCredentials(email, it) }
        }
    }

    suspend fun saveToken(token: String, userId: String, email: String) {
        context.dataStore.edit { prefs ->
            prefs[TOKEN_KEY] = token
            prefs[USER_ID_KEY] = userId
            prefs[EMAIL_KEY] = email
        }
    }

    suspend fun clearToken() {
        context.dataStore.edit { prefs ->
            prefs.remove(TOKEN_KEY)
            prefs.remove(USER_ID_KEY)
            prefs.remove(EMAIL_KEY)
        }
    }

    /** 勾选「记住密码」时调用；加密失败则静默放弃，不影响登录本身。 */
    suspend fun saveCredentials(email: String, password: String) {
        val encrypted = runCatching { CredentialCipher.encrypt(password) }.getOrNull() ?: return
        context.dataStore.edit { prefs ->
            prefs[SAVED_EMAIL_KEY] = email
            prefs[SAVED_PASSWORD_KEY] = encrypted
        }
    }

    suspend fun clearCredentials() {
        context.dataStore.edit { prefs ->
            prefs.remove(SAVED_EMAIL_KEY)
            prefs.remove(SAVED_PASSWORD_KEY)
        }
    }
}

/** 本地记住的账号密码，仅用于登录页自动填充。 */
data class SavedCredentials(val email: String, val password: String)
