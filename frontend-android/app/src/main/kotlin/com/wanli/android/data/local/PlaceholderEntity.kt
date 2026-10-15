package com.wanli.android.data.local

import androidx.room.Entity
import androidx.room.PrimaryKey

/**
 * 占位 Room 实体，阶段 1 仅使用 DataStore 缓存 Token。
 * 后续阶段替换为真正的本地数据实体。
 */
@Entity(tableName = "placeholder")
data class PlaceholderEntity(
    @PrimaryKey val id: Long = 0
)
