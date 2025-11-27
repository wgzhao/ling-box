package com.lingbox

import org.junit.jupiter.api.Test
import org.junit.jupiter.api.Assertions.*

class BcryptUtilsTest {
    @Test
    fun `hash generates valid bcrypt hash`() {
        val password = "testpassword123"
        val hash = BcryptUtils.hash(password)
        
        assertTrue(hash.startsWith("\$2a\$"))
        assertEquals(60, hash.length)
    }

    @Test
    fun `verify returns true for correct password`() {
        val password = "testpassword123"
        val hash = BcryptUtils.hash(password)
        
        assertTrue(BcryptUtils.verify(password, hash))
    }

    @Test
    fun `verify returns false for incorrect password`() {
        val password = "testpassword123"
        val hash = BcryptUtils.hash(password)
        
        assertFalse(BcryptUtils.verify("wrongpassword", hash))
    }

    @Test
    fun `hash with different costs produces different hashes`() {
        val password = "testpassword123"
        val hash4 = BcryptUtils.hash(password, 4)
        val hash10 = BcryptUtils.hash(password, 10)
        
        assertNotEquals(hash4, hash10)
        assertTrue(BcryptUtils.verify(password, hash4))
        assertTrue(BcryptUtils.verify(password, hash10))
    }

    @Test
    fun `hash same password multiple times produces different hashes`() {
        val password = "testpassword123"
        val hash1 = BcryptUtils.hash(password)
        val hash2 = BcryptUtils.hash(password)
        
        assertNotEquals(hash1, hash2)
        assertTrue(BcryptUtils.verify(password, hash1))
        assertTrue(BcryptUtils.verify(password, hash2))
    }
}
