package com.lingbox

import org.junit.jupiter.api.Test
import org.junit.jupiter.api.Assertions.*

class UrlUtilsTest {
    @Test
    fun `encode simple string`() {
        val result = UrlUtils.encode("hello world")
        assertEquals("hello+world", result)
    }

    @Test
    fun `encode string with special characters`() {
        val result = UrlUtils.encode("hello&world=test")
        assertEquals("hello%26world%3Dtest", result)
    }

    @Test
    fun `encode Chinese characters`() {
        val result = UrlUtils.encode("你好")
        assertEquals("%E4%BD%A0%E5%A5%BD", result)
    }

    @Test
    fun `decode simple string`() {
        val result = UrlUtils.decode("hello+world")
        assertEquals("hello world", result)
    }

    @Test
    fun `decode string with special characters`() {
        val result = UrlUtils.decode("hello%26world%3Dtest")
        assertEquals("hello&world=test", result)
    }

    @Test
    fun `decode Chinese characters`() {
        val result = UrlUtils.decode("%E4%BD%A0%E5%A5%BD")
        assertEquals("你好", result)
    }

    @Test
    fun `encode and decode round trip`() {
        val original = "Hello World! 你好世界 @#$%"
        val encoded = UrlUtils.encode(original)
        val decoded = UrlUtils.decode(encoded)
        assertEquals(original, decoded)
    }
}
