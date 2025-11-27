package com.lingbox

import org.junit.jupiter.api.Test
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.io.TempDir
import java.io.File
import java.nio.file.Path
import javax.imageio.ImageIO

class QrCodeUtilsTest {
    @TempDir
    lateinit var tempDir: Path

    @Test
    fun `generate creates QR code file`() {
        val outputFile = tempDir.resolve("test-qr.png").toFile()
        
        QrCodeUtils.generate("https://example.com", outputFile.absolutePath)
        
        assertTrue(outputFile.exists())
        assertTrue(outputFile.length() > 0)
    }

    @Test
    fun `generate creates QR code with correct size`() {
        val outputFile = tempDir.resolve("test-qr-sized.png").toFile()
        val expectedSize = 400
        
        QrCodeUtils.generate("https://example.com", outputFile.absolutePath, size = expectedSize)
        
        val image = ImageIO.read(outputFile)
        assertEquals(expectedSize, image.width)
        assertEquals(expectedSize, image.height)
    }

    @Test
    fun `generate supports different formats`() {
        val pngFile = tempDir.resolve("test-qr.png").toFile()
        val jpgFile = tempDir.resolve("test-qr.jpg").toFile()
        
        QrCodeUtils.generate("test", pngFile.absolutePath, format = "PNG")
        QrCodeUtils.generate("test", jpgFile.absolutePath, format = "JPG")
        
        assertTrue(pngFile.exists())
        assertTrue(jpgFile.exists())
    }

    @Test
    fun `generateToFile works correctly`() {
        val outputFile = tempDir.resolve("test-qr-file.png").toFile()
        
        QrCodeUtils.generateToFile("https://example.com", outputFile)
        
        assertTrue(outputFile.exists())
    }

    @Test
    fun `generate handles Chinese characters`() {
        val outputFile = tempDir.resolve("test-qr-chinese.png").toFile()
        
        QrCodeUtils.generate("你好世界", outputFile.absolutePath)
        
        assertTrue(outputFile.exists())
        assertTrue(outputFile.length() > 0)
    }
}
