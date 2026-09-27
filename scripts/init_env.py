#!/usr/bin/env python3
"""Create credentials once. Never overwrite or print an existing password."""
import os
import secrets
from common import ROOT

def main():
    path = ROOT / ".env"
    if path.exists():
        print(".env already exists; credentials left unchanged")
        return
    text = (ROOT / ".env.example").read_text()
    while "REPLACE_WITH_RANDOM_HEX" in text:
        text = text.replace("REPLACE_WITH_RANDOM_HEX", secrets.token_hex(24), 1)
    fd = os.open(path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o600)
    with os.fdopen(fd, "w") as out:
        out.write(text)
    print("Created private .env; demo user: devops. Read API_PASSWORD locally when signing in.")
if __name__ == "__main__": main()
