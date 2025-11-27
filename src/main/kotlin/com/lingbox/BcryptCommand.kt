package com.lingbox

import at.favre.lib.crypto.bcrypt.BCrypt
import picocli.CommandLine.Command
import picocli.CommandLine.Option
import picocli.CommandLine.Parameters

@Command(
    name = "bcrypt",
    mixinStandardHelpOptions = true,
    description = ["BCrypt password hashing tool"]
)
class BcryptCommand : Runnable {
    @Option(names = ["-g", "--generate"], description = ["Generate a bcrypt hash"])
    var generate: Boolean = false

    @Option(names = ["-v", "--verify"], description = ["Verify a password against a hash"])
    var verify: Boolean = false

    @Option(names = ["-c", "--cost"], description = ["Cost factor for bcrypt (default: 12)"])
    var cost: Int = 12

    @Parameters(index = "0", description = ["The password to hash or verify"])
    lateinit var password: String

    @Parameters(index = "1", arity = "0..1", description = ["The hash to verify against (required for -v/--verify)"])
    var hash: String? = null

    override fun run() {
        when {
            generate -> println(BcryptUtils.hash(password, cost))
            verify -> {
                if (hash == null) {
                    println("Error: Hash is required for verification")
                    return
                }
                val result = BcryptUtils.verify(password, hash!!)
                println(if (result) "Password matches!" else "Password does not match.")
            }
            else -> {
                println("Please specify -g/--generate or -v/--verify")
            }
        }
    }
}

object BcryptUtils {
    fun hash(password: String, cost: Int = 12): String {
        return BCrypt.withDefaults().hashToString(cost, password.toCharArray())
    }

    fun verify(password: String, hash: String): Boolean {
        return BCrypt.verifyer().verify(password.toCharArray(), hash).verified
    }
}
