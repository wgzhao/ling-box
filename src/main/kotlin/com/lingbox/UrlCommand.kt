package com.lingbox

import picocli.CommandLine.Command
import picocli.CommandLine.Option
import picocli.CommandLine.Parameters
import java.net.URLDecoder
import java.net.URLEncoder
import java.nio.charset.StandardCharsets

@Command(
    name = "url",
    mixinStandardHelpOptions = true,
    description = ["URL encoding and decoding tool"]
)
class UrlCommand : Runnable {
    @Option(names = ["-e", "--encode"], description = ["Encode the input string"])
    var encode: Boolean = false

    @Option(names = ["-d", "--decode"], description = ["Decode the input string"])
    var decode: Boolean = false

    @Parameters(index = "0", description = ["The string to encode or decode"])
    lateinit var input: String

    override fun run() {
        when {
            encode -> println(UrlUtils.encode(input))
            decode -> println(UrlUtils.decode(input))
            else -> {
                println("Please specify -e/--encode or -d/--decode")
            }
        }
    }
}

object UrlUtils {
    fun encode(input: String): String {
        return URLEncoder.encode(input, StandardCharsets.UTF_8)
    }

    fun decode(input: String): String {
        return URLDecoder.decode(input, StandardCharsets.UTF_8)
    }
}
