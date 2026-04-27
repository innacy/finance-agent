"""Helper script for setting up Telegram bot.

Usage: python -m scripts.setup_telegram
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def main():
    print("=" * 60)
    print("  Telegram Bot Setup Guide")
    print("=" * 60)
    print()
    print("Step 1: Create a new bot")
    print("  1. Open Telegram and search for @BotFather")
    print("  2. Send /newbot")
    print("  3. Choose a name (e.g., 'My Finance Agent')")
    print("  4. Choose a username (e.g., 'my_finance_bot')")
    print("  5. Copy the bot token")
    print()
    print("Step 2: Get your Chat ID")
    print("  1. Send any message to your new bot")
    print("  2. Open: https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates")
    print("  3. Find 'chat':{'id': XXXXXXX} in the response")
    print("  4. Copy the chat ID number")
    print()
    print("Step 3: Configure the agent")
    print("  Add these to your .env file:")
    print()

    token = input("  Enter bot token (or press Enter to skip): ").strip()
    chat_id = input("  Enter chat ID (or press Enter to skip): ").strip()

    if token or chat_id:
        env_path = os.path.join(os.path.dirname(os.path.dirname(__file__)), ".env")
        lines = []
        if os.path.exists(env_path):
            with open(env_path, "r") as f:
                lines = f.readlines()

        updated = {}
        if token:
            updated["TELEGRAM_BOT_TOKEN"] = token
        if chat_id:
            updated["TELEGRAM_CHAT_ID"] = chat_id

        new_lines = []
        for line in lines:
            key = line.split("=")[0].strip() if "=" in line else ""
            if key in updated:
                new_lines.append(f"{key}={updated.pop(key)}\n")
            else:
                new_lines.append(line)

        for key, value in updated.items():
            new_lines.append(f"{key}={value}\n")

        with open(env_path, "w") as f:
            f.writelines(new_lines)

        print(f"\n  ✅ Updated {env_path}")
    else:
        print("\n  Skipped. Manually add TELEGRAM_BOT_TOKEN and TELEGRAM_CHAT_ID to .env")

    print("\nDone! The bot will send reports after each sync run.")


if __name__ == "__main__":
    main()
