package com.lingbox

import com.google.zxing.BarcodeFormat
import com.google.zxing.EncodeHintType
import com.google.zxing.client.j2se.MatrixToImageWriter
import com.google.zxing.qrcode.QRCodeWriter
import com.google.zxing.qrcode.decoder.ErrorCorrectionLevel
import picocli.CommandLine.Command
import picocli.CommandLine.Option
import picocli.CommandLine.Parameters
import java.io.File
import java.nio.file.Paths

@Command(
    name = "qrcode",
    mixinStandardHelpOptions = true,
    description = ["QR Code generation tool"]
)
class QrCodeCommand : Runnable {
    @Parameters(index = "0", description = ["The text to encode in the QR code"])
    lateinit var text: String

    @Option(names = ["-o", "--output"], description = ["Output file path (default: qrcode.png)"])
    var output: String = "qrcode.png"

    @Option(names = ["-s", "--size"], description = ["Size of the QR code in pixels (default: 300)"])
    var size: Int = 300

    @Option(names = ["-f", "--format"], description = ["Image format: PNG, JPG, GIF (default: PNG)"])
    var format: String = "PNG"

    override fun run() {
        try {
            QrCodeUtils.generate(text, output, size, format)
            println("QR code generated successfully: $output")
        } catch (e: Exception) {
            println("Error generating QR code: ${e.message}")
        }
    }
}

object QrCodeUtils {
    fun generate(text: String, outputPath: String, size: Int = 300, format: String = "PNG") {
        val hints = mapOf(
            EncodeHintType.CHARACTER_SET to "UTF-8",
            EncodeHintType.ERROR_CORRECTION to ErrorCorrectionLevel.H,
            EncodeHintType.MARGIN to 1
        )

        val qrCodeWriter = QRCodeWriter()
        val bitMatrix = qrCodeWriter.encode(text, BarcodeFormat.QR_CODE, size, size, hints)
        
        val path = Paths.get(outputPath)
        MatrixToImageWriter.writeToPath(bitMatrix, format, path)
    }

    fun generateToFile(text: String, file: File, size: Int = 300, format: String = "PNG") {
        generate(text, file.absolutePath, size, format)
    }
}
