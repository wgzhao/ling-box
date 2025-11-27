package com.lingbox

import picocli.CommandLine
import picocli.CommandLine.Command
import kotlin.system.exitProcess

@Command(
    name = "ling-box",
    mixinStandardHelpOptions = true,
    version = ["ling-box 1.0.0"],
    description = ["玲珑盒 - A collection of useful utility tools"],
    subcommands = [
        UrlCommand::class,
        Base64Command::class,
        BcryptCommand::class,
        QrCodeCommand::class,
        PasswordCommand::class
    ]
)
class LingBoxApp : Runnable {
    override fun run() {
        CommandLine(this).usage(System.out)
    }
}

fun main(args: Array<String>) {
    val exitCode = CommandLine(LingBoxApp()).execute(*args)
    exitProcess(exitCode)
}
