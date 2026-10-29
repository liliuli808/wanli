package com.wanli.android.presentation.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.outlined.SoupKitchen
import androidx.compose.material3.Button
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.key
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.wanli.android.presentation.components.StatusBanner
import com.wanli.android.presentation.components.StatusBannerType
import com.wanli.android.presentation.components.WanLiCard
import com.wanli.android.presentation.components.cardContentPadding
import com.wanli.android.presentation.theme.OnTomato
import com.wanli.android.presentation.theme.TomatoRed
import com.wanli.android.presentation.theme.TomatoSoft

private enum class LoginMode { Login, Register }

@Composable
fun LoginScreen(
    viewModel: LoginViewModel = androidx.hilt.navigation.compose.hiltViewModel(),
    onLoggedIn: () -> Unit
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    var email by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var turnstileToken by remember { mutableStateOf<String?>(null) }
    var mode by remember { mutableStateOf(LoginMode.Login) }
    var widgetEpoch by remember { mutableStateOf(0) }

    LaunchedEffect(uiState) {
        if (uiState is LoginUiState.LoggedIn) {
            onLoggedIn()
        }
    }

    // 登录/注册失败后，强制刷新验证码（token 一次性）
    LaunchedEffect(uiState) {
        if (uiState is LoginUiState.Error) {
            turnstileToken = null
            widgetEpoch++
        }
    }

    fun switchMode() {
        mode = if (mode == LoginMode.Login) LoginMode.Register else LoginMode.Login
        turnstileToken = null
        widgetEpoch++
    }

    val submit: () -> Unit = {
        turnstileToken?.let { token ->
            if (mode == LoginMode.Login) {
                viewModel.login(email, password, token)
            } else {
                viewModel.register(email, password, token)
            }
        }
    }

    Scaffold { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .verticalScroll(rememberScrollState())
                .imePadding()
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .height(200.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                Box(
                    contentAlignment = Alignment.Center,
                    modifier = Modifier
                        .size(96.dp)
                        .clip(CircleShape)
                        .background(TomatoRed)
                ) {
                    Icon(
                        imageVector = Icons.Outlined.SoupKitchen,
                        contentDescription = null,
                        tint = OnTomato,
                        modifier = Modifier.size(48.dp)
                    )
                }

                Text(
                    text = "碗里",
                    style = MaterialTheme.typography.headlineMedium,
                    modifier = Modifier.padding(top = 16.dp)
                )
                Text(
                    text = "把视频变成能下锅的菜谱",
                    style = MaterialTheme.typography.bodyMedium,
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    textAlign = TextAlign.Center,
                    modifier = Modifier.padding(top = 4.dp)
                )
            }

            WanLiCard(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.cardContentPadding()) {
                    OutlinedTextField(
                        value = email,
                        onValueChange = { email = it },
                        label = { Text("邮箱") },
                        singleLine = true,
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Email),
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.fillMaxWidth()
                    )

                    Spacer(modifier = Modifier.height(8.dp))

                    OutlinedTextField(
                        value = password,
                        onValueChange = { password = it },
                        label = { Text("密码") },
                        singleLine = true,
                        visualTransformation = PasswordVisualTransformation(),
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                        shape = RoundedCornerShape(16.dp),
                        modifier = Modifier.fillMaxWidth()
                    )

                    Spacer(modifier = Modifier.height(16.dp))

                    key(widgetEpoch) {
                        TurnstileWidget(
                            widgetUrl = viewModel.captchaUrl,
                            onTokenReceived = { turnstileToken = it },
                            onError = { turnstileToken = null },
                            modifier = Modifier
                                .fillMaxWidth()
                                .height(80.dp)
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            Button(
                onClick = submit,
                modifier = Modifier.fillMaxWidth(),
                shape = MaterialTheme.shapes.large,
                enabled = email.isNotBlank() &&
                    password.isNotBlank() &&
                    turnstileToken != null &&
                    uiState !is LoginUiState.Loading
            ) {
                Text(if (mode == LoginMode.Login) "登录" else "注册")
            }

            TextButton(onClick = ::switchMode) {
                Text(if (mode == LoginMode.Login) "没有账号？去注册" else "已有账号？去登录")
            }

            if (uiState is LoginUiState.Error) {
                Spacer(modifier = Modifier.height(16.dp))
                StatusBanner(type = StatusBannerType.ERROR) {
                    Text(
                        text = (uiState as LoginUiState.Error).message,
                        style = MaterialTheme.typography.bodyMedium
                    )
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            Text(
                text = "登录即表示同意用户协议与隐私政策",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                textAlign = TextAlign.Center,
                modifier = Modifier.fillMaxWidth()
            )
        }
    }
}
