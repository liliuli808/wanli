package com.wanli.android.domain.model

/** 服务端返回的最新安装包信息。 */
data class AppUpdateInfo(
    val versionCode: Int,
    val versionName: String,
    /** 安装包字节数，未知时为 0。 */
    val size: Long,
    /** 安装包 SHA-256，用于下载后校验；为空表示服务端未提供。 */
    val sha256: String,
    val commit: String,
    /** APK 下载地址（已解析为绝对地址）。 */
    val downloadUrl: String,
)
