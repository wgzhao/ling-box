package com.lingbox

import picocli.CommandLine.Command
import picocli.CommandLine.Option
import picocli.CommandLine.Parameters
import java.util.Base64
import java.nio.charset.StandardCharsets

@Command(
    name = "base64",
    mixinStandardHelpOptions = true,
    description = ["Base64 encoding and decoding tool"]
)
class Base64Command : Runnable {
    @Option(names = ["-e", "--encode"], description = ["Encode the input string"])
    var encode: Boolean = false

    @Option(names = ["-d", "--decode"], description = ["Decode the input string"])
    var decode: Boolean = false

    @Option(names = ["-u", "--url-safe"], description = ["Use URL-safe Base64 encoding"])
    var urlSafe: Boolean = false

    @Parameters(index = "0", description = ["The string to encode or decode"])
    lateinit var input: String

    override fun run() {
        when {
            encode -> println(Base64Utils.encode(input, urlSafe))
            decode -> println(Base64Utils.decode(input, urlSafe))
            else -> {
                println("Please specify -e/--encode or -d/--decode")
            }
        }
    }
}

object Base64Utils {
    fun encode(input: String, urlSafe: Boolean = false): String {
        val encoder = if (urlSafe) Base64.getUrlEncoder() else Base64.getEncoder()
        return encoder.encodeToString(input.toByteArray(StandardCharsets.UTF_8))
    }

    fun decode(input: String, urlSafe: Boolean = false): String {
        val decoder = if (urlSafe) Base64.getUrlDecoder() else Base64.getDecoder()
        return String(decoder.decode(input), StandardCharsets.UTF_8)
    }
}
