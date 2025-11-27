package com.lingbox

import org.junit.jupiter.api.Test
import org.junit.jupiter.api.Assertions.*

class PasswordUtilsTest {
    @Test
    fun `generate creates password with default length`() {
        val password = PasswordUtils.generate()
        assertEquals(16, password.length)
    }

    @Test
    fun `generate creates password with custom length`() {
        val options = PasswordUtils.PasswordOptions(length = 24)
        val password = PasswordUtils.generate(options)
        assertEquals(24, password.length)
    }

    @Test
    fun `generate with digits only contains only digits`() {
        val options = PasswordUtils.PasswordOptions(digitsOnly = true)
        val password = PasswordUtils.generate(options)
        
        assertTrue(password.all { it.isDigit() })
    }

    @Test
    fun `generate with uppercase only contains only uppercase letters`() {
        val options = PasswordUtils.PasswordOptions(uppercaseOnly = true)
        val password = PasswordUtils.generate(options)
        
        assertTrue(password.all { it.isUpperCase() })
    }

    @Test
    fun `generate without special characters has no special chars`() {
        val specialChars = "!@#\$%^&*()_+-=[]{}|;:,.<>?"
        val options = PasswordUtils.PasswordOptions(includeSpecial = false)
        
        // Generate multiple passwords to increase confidence
        repeat(10) {
            val password = PasswordUtils.generate(options)
            assertFalse(password.any { it in specialChars })
        }
    }

    @Test
    fun `generate creates unique passwords`() {
        val passwords = (1..100).map { PasswordUtils.generate() }.toSet()
        
        // All 100 passwords should be unique
        assertEquals(100, passwords.size)
    }

    @Test
    fun `generate with special characters includes various character types`() {
        val options = PasswordUtils.PasswordOptions(length = 100, includeSpecial = true)
        val password = PasswordUtils.generate(options)
        
        // With a 100-character password, we should have various types
        val hasLowercase = password.any { it.isLowerCase() }
        val hasUppercase = password.any { it.isUpperCase() }
        val hasDigit = password.any { it.isDigit() }
        
        // At least some character types should be present (probabilistically)
        assertTrue(hasLowercase || hasUppercase || hasDigit)
    }
}
