package com.lingbox

import org.junit.jupiter.api.Test
import org.junit.jupiter.api.Assertions.*

class Base64UtilsTest {
    @Test
    fun `encode simple string`() {
        val result = Base64Utils.encode("hello")
        assertEquals("aGVsbG8=", result)
    }

    @Test
    fun `decode simple string`() {
        val result = Base64Utils.decode("aGVsbG8=")
        assertEquals("hello", result)
    }

    @Test
    fun `encode Chinese characters`() {
        val result = Base64Utils.encode("你好")
        assertEquals("5L2g5aW9", result)
    }

    @Test
    fun `decode Chinese characters`() {
        val result = Base64Utils.decode("5L2g5aW9")
        assertEquals("你好", result)
    }

    @Test
    fun `encode and decode round trip`() {
        val original = "Hello World! 你好世界"
        val encoded = Base64Utils.encode(original)
        val decoded = Base64Utils.decode(encoded)
        assertEquals(original, decoded)
    }

    @Test
    fun `url-safe encode`() {
        val input = "test+/string"
        val result = Base64Utils.encode(input, urlSafe = true)
        assertFalse(result.contains("+"))
        assertFalse(result.contains("/"))
    }

    @Test
    fun `url-safe encode and decode round trip`() {
        val original = "test+/string?param=value"
        val encoded = Base64Utils.encode(original, urlSafe = true)
        val decoded = Base64Utils.decode(encoded, urlSafe = true)
        assertEquals(original, decoded)
    }
}
