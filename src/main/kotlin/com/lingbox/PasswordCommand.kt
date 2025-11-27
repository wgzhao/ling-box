package com.lingbox

import picocli.CommandLine.Command
import picocli.CommandLine.Option
import java.security.SecureRandom

@Command(
    name = "password",
    mixinStandardHelpOptions = true,
    description = ["Secure password generation tool"]
)
class PasswordCommand : Runnable {
    @Option(names = ["-l", "--length"], description = ["Password length (default: 16)"])
    var length: Int = 16

    @Option(names = ["-n", "--no-special"], description = ["Exclude special characters"])
    var noSpecial: Boolean = false

    @Option(names = ["-u", "--uppercase-only"], description = ["Use only uppercase letters"])
    var uppercaseOnly: Boolean = false

    @Option(names = ["-d", "--digits-only"], description = ["Use only digits"])
    var digitsOnly: Boolean = false

    @Option(names = ["-c", "--count"], description = ["Number of passwords to generate (default: 1)"])
    var count: Int = 1

    override fun run() {
        repeat(count) {
            val options = PasswordUtils.PasswordOptions(
                length = length,
                includeSpecial = !noSpecial,
                uppercaseOnly = uppercaseOnly,
                digitsOnly = digitsOnly
            )
            println(PasswordUtils.generate(options))
        }
    }
}

object PasswordUtils {
    private const val LOWERCASE = "abcdefghijklmnopqrstuvwxyz"
    private const val UPPERCASE = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
    private const val DIGITS = "0123456789"
    private const val SPECIAL = "!@#$%^&*()_+-=[]{}|;:,.<>?"

    data class PasswordOptions(
        val length: Int = 16,
        val includeSpecial: Boolean = true,
        val uppercaseOnly: Boolean = false,
        val digitsOnly: Boolean = false
    )

    fun generate(options: PasswordOptions = PasswordOptions()): String {
        val charset = when {
            options.digitsOnly -> DIGITS
            options.uppercaseOnly -> UPPERCASE
            options.includeSpecial -> LOWERCASE + UPPERCASE + DIGITS + SPECIAL
            else -> LOWERCASE + UPPERCASE + DIGITS
        }

        val random = SecureRandom()
        return (1..options.length)
            .map { charset[random.nextInt(charset.length)] }
            .joinToString("")
    }
}
