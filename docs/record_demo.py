#!/usr/bin/env python3
"""Record the ling-box demo session to docs/demo.cast with asciinema.

Usage: python3 docs/record_demo.py
Requires: asciinema 3.x (set $ASCIINEMA to its location if not on PATH).

asciinema runs on a pseudo-terminal of 120x40 cells so the commands format
themselves for that width. The script types each command character by
character and pauses between commands so the pacing survives in the
recorded cast file. Render the result with docs/cast2gif.py.
"""
import fcntl
import os
import pty
import struct
import sys
import termios
import time

HERE = os.path.dirname(os.path.abspath(__file__))
REPO = os.path.dirname(HERE)
OUT = os.path.join(HERE, "demo.cast")

COMMANDS = [
    "lingbox ipgeo 8.8.8.8",
    "lingbox ipcalc 192.168.0.1/24",
    "lingbox uuid -n 5",
    "lingbox date diff 2026-08-28 2027-01-01",
    "lingbox qrcode 'https://github.com/wgzhao/ling-box' -o /tmp/lingbox-qr.png",
    "lingbox imgcat /tmp/lingbox-qr.png --renderer halfblock -w 60",
]

TYPE_DELAY = 0.04  # seconds per character
PAUSE = 2.2        # seconds after each command runs


def drain(master: int) -> None:
    """Read and discard pty output so the buffer never fills up."""
    try:
        while True:
            chunk = os.read(master, 4096)
            if not chunk:
                break
    except (BlockingIOError, OSError):
        pass


def main() -> int:
    asciinema = os.environ.get("ASCIINEMA", "asciinema")
    env = dict(os.environ)
    env["PATH"] = os.pathsep.join([REPO, os.path.dirname(asciinema), env.get("PATH", "")])
    # TERM=dumb keeps fish from probing the terminal on startup: the probe
    # waits for replies that a recording pty never sends, so the session
    # would hang. Command output colors are unaffected (they check isatty).
    env["TERM"] = "dumb"
    pid, master = pty.fork()
    if pid == 0:  # child: the asciinema session
        # --no-config skips the user's fish setup (starship etc.) so the
        # demo shows a clean stock prompt; -C suppresses the greeting.
        os.execvpe(asciinema, [asciinema, "rec", "--overwrite",
                               "-c", "fish --no-config -C 'set -g fish_greeting'", OUT], env)
    # parent: fix the pty size and drive the session
    fcntl.ioctl(master, termios.TIOCSWINSZ, struct.pack("HHHH", 40, 120, 0, 0))
    os.set_blocking(master, False)

    def send(text: str) -> None:
        for ch in text:
            os.write(master, ch.encode())
            time.sleep(TYPE_DELAY)
        os.write(master, b"\n")
        drain(master)

    time.sleep(1.5)  # let the prompt settle
    drain(master)
    for cmd in COMMANDS:
        send(cmd)
        time.sleep(PAUSE)
        drain(master)
    os.write(master, b"exit\n")
    # Keep draining while asciinema winds down: it prints its goodbye
    # message to the pty and blocks if the buffer fills up.
    deadline = time.monotonic() + 5
    while time.monotonic() < deadline:
        drain(master)
        done, status = os.waitpid(pid, os.WNOHANG)
        if done:
            return os.waitstatus_to_exitcode(status)
        time.sleep(0.1)
    try:
        os.kill(pid, 9)
    except ProcessLookupError:
        pass
    return 1


if __name__ == "__main__":
    sys.exit(main())
