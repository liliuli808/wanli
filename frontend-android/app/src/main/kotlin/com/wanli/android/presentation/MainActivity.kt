package com.wanli.android.presentation

import android.os.Bundle
import android.widget.Toast
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.enableEdgeToEdge
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.Surface
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import androidx.navigation.compose.rememberNavController
import com.wanli.android.presentation.navigation.WanLiNavHost
import com.wanli.android.presentation.theme.WanLiTheme
import com.wanli.android.presentation.update.ApkInstaller
import com.wanli.android.presentation.update.AppUpdateDialog
import com.wanli.android.presentation.update.AppUpdateUiState
import com.wanli.android.presentation.update.AppUpdateViewModel
import dagger.hilt.android.AndroidEntryPoint

@AndroidEntryPoint
class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        enableEdgeToEdge()
        setContent {
            WanLiTheme {
                Surface(modifier = Modifier.fillMaxSize()) {
                    val navController = rememberNavController()
                    val context = LocalContext.current
                    val updateViewModel: AppUpdateViewModel = hiltViewModel()
                    val updateState by updateViewModel.uiState.collectAsStateWithLifecycle()

                    // 每次冷启动检查一次；同一版本被忽略后 24 小时内不再提示
                    LaunchedEffect(Unit) { updateViewModel.checkOnLaunch() }

                    WanLiNavHost(
                        navController = navController,
                        onCheckForUpdate = { updateViewModel.check(manual = true) },
                    )

                    AppUpdateDialog(
                        state = updateState,
                        onDismiss = updateViewModel::dismiss,
                        onDownload = updateViewModel::download,
                        onInstall = {
                            val apk = (updateState as? AppUpdateUiState.Ready)?.apk
                            if (apk != null) {
                                if (ApkInstaller.canInstall(context)) {
                                    ApkInstaller.install(context, apk)
                                    updateViewModel.dismiss()
                                } else {
                                    // Android 8.0+ 需要用户先授予「安装未知应用」权限
                                    ApkInstaller.requestInstallPermission(context)
                                    Toast.makeText(
                                        context,
                                        "请允许「碗里」安装应用，然后返回重试",
                                        Toast.LENGTH_LONG,
                                    ).show()
                                }
                            }
                        },
                    )
                }
            }
        }
    }
}
