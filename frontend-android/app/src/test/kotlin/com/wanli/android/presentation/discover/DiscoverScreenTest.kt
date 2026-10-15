package com.wanli.android.presentation.discover

import com.wanli.android.data.remote.RecipeSummaryDto
import org.junit.Assert.assertNotEquals
import org.junit.Test

class DiscoverScreenTest {

    @Test
    fun `list keys remain unique when recipe ids are duplicated`() {
        val first = summary("version-1")
        val second = summary("version-2")

        assertNotEquals(discoverItemKey(first), discoverItemKey(second))
    }

    private fun summary(versionId: String) = RecipeSummaryDto(
        recipe_id = "recipe-1",
        title = "红烧肉",
        recipe_version_id = versionId
    )
}
