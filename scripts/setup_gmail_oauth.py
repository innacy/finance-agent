"""One-time script to set up Gmail OAuth2 credentials.

Usage:
    1. Go to https://console.cloud.google.com/apis/credentials
    2. Create OAuth 2.0 Client ID (Desktop app)
    3. Download the JSON and save as data/gmail_credentials.json
    4. Run: python -m scripts.setup_gmail_oauth
    5. Follow the browser prompt to authorize
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from config.settings import settings
from google_auth_oauthlib.flow import InstalledAppFlow


def main():
    creds_path = settings.gmail_credentials_path
    token_path = settings.gmail_token_path

    if not os.path.exists(creds_path):
        print(f"ERROR: Gmail credentials file not found at: {creds_path}")
        print()
        print("Steps to get credentials:")
        print("1. Go to https://console.cloud.google.com/apis/credentials")
        print("2. Create a project (or select existing)")
        print("3. Enable the Gmail API")
        print("4. Create OAuth 2.0 Client ID -> Desktop application")
        print("5. Download the JSON file")
        print(f"6. Save it as: {creds_path}")
        print("7. Run this script again")
        sys.exit(1)

    print("Starting Gmail OAuth2 authorization flow...")
    print("A browser window will open. Sign in and grant read-only access.")
    print()

    flow = InstalledAppFlow.from_client_secrets_file(
        creds_path, settings.gmail_scopes
    )
    creds = flow.run_local_server(port=0)

    os.makedirs(os.path.dirname(token_path), exist_ok=True)
    with open(token_path, "w") as f:
        f.write(creds.to_json())

    print(f"\nAuthorization successful! Token saved to: {token_path}")
    print("The finance agent can now read your Gmail.")


if __name__ == "__main__":
    main()
