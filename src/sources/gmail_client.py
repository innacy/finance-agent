"""Gmail API client for fetching financial emails with OAuth2 and incremental sync."""

import base64
import logging
import os
from datetime import datetime, timedelta
from email.utils import parsedate_to_datetime
from typing import Optional

from google.auth.transport.requests import Request
from google.oauth2.credentials import Credentials
from google_auth_oauthlib.flow import InstalledAppFlow
from googleapiclient.discovery import build

from config.bank_templates import build_gmail_query
from config.settings import settings
from src.db.mongo import get_sync_state, update_sync_state

logger = logging.getLogger(__name__)


class GmailClient:
    def __init__(self):
        self._service = None
        self._creds: Optional[Credentials] = None

    def _authenticate(self) -> Credentials:
        creds = None
        token_path = settings.gmail_token_path

        if os.path.exists(token_path):
            creds = Credentials.from_authorized_user_file(
                token_path, settings.gmail_scopes
            )

        if not creds or not creds.valid:
            if creds and creds.expired and creds.refresh_token:
                creds.refresh(Request())
            else:
                if not os.path.exists(settings.gmail_credentials_path):
                    raise FileNotFoundError(
                        f"Gmail credentials not found at {settings.gmail_credentials_path}. "
                        "Run `python -m scripts.setup_gmail_oauth` first."
                    )
                flow = InstalledAppFlow.from_client_secrets_file(
                    settings.gmail_credentials_path, settings.gmail_scopes
                )
                creds = flow.run_local_server(port=0)

            os.makedirs(os.path.dirname(token_path), exist_ok=True)
            with open(token_path, "w") as f:
                f.write(creds.to_json())

        return creds

    @property
    def service(self):
        if self._service is None:
            self._creds = self._authenticate()
            self._service = build("gmail", "v1", credentials=self._creds)
        return self._service

    def fetch_financial_emails(self, user_id: str = "default",
                                max_results: int = 100,
                                days_back: Optional[int] = None) -> list[dict]:
        """Fetch financial emails since last sync (incremental).

        Returns list of parsed email dicts with: id, subject, from, date, body_text, body_html, attachments.
        """
        sync = get_sync_state(user_id, "gmail")

        query_parts = [build_gmail_query()]

        if days_back:
            after_date = (datetime.utcnow() - timedelta(days=days_back)).strftime("%Y/%m/%d")
            query_parts.append(f"after:{after_date}")
        elif sync and sync.get("last_sync_time"):
            after_date = sync["last_sync_time"].strftime("%Y/%m/%d")
            query_parts.append(f"after:{after_date}")

        query = " ".join(query_parts)
        logger.info("Gmail query: %s", query)

        messages = []
        page_token = None

        while True:
            resp = self.service.users().messages().list(
                userId="me", q=query, maxResults=min(max_results, 100),
                pageToken=page_token
            ).execute()

            if "messages" not in resp:
                break

            for msg_ref in resp["messages"]:
                if sync and sync.get("last_email_id") == msg_ref["id"]:
                    return messages

                msg = self._get_message(msg_ref["id"])
                if msg:
                    messages.append(msg)

                if len(messages) >= max_results:
                    break

            page_token = resp.get("nextPageToken")
            if not page_token or len(messages) >= max_results:
                break

        if messages:
            update_sync_state(
                user_id, "gmail",
                last_email_id=messages[0]["id"],
                last_sync_time=datetime.utcnow(),
            )
            logger.info("Fetched %d financial emails", len(messages))

        return messages

    def _get_message(self, msg_id: str) -> Optional[dict]:
        try:
            msg = self.service.users().messages().get(
                userId="me", id=msg_id, format="full"
            ).execute()

            headers = {h["name"].lower(): h["value"] for h in msg["payload"]["headers"]}

            date_str = headers.get("date", "")
            try:
                date = parsedate_to_datetime(date_str)
            except Exception:
                date = datetime.utcnow()

            body_text, body_html = self._extract_body(msg["payload"])

            attachments = []
            self._extract_attachments(msg["payload"], msg_id, attachments)

            return {
                "id": msg_id,
                "subject": headers.get("subject", ""),
                "from": headers.get("from", ""),
                "to": headers.get("to", ""),
                "date": date,
                "body_text": body_text,
                "body_html": body_html,
                "snippet": msg.get("snippet", ""),
                "attachments": attachments,
            }
        except Exception:
            logger.exception("Failed to fetch message %s", msg_id)
            return None

    def _extract_body(self, payload: dict) -> tuple[str, str]:
        text, html = "", ""
        if payload.get("mimeType") == "text/plain" and payload.get("body", {}).get("data"):
            text = base64.urlsafe_b64decode(payload["body"]["data"]).decode("utf-8", errors="replace")
        elif payload.get("mimeType") == "text/html" and payload.get("body", {}).get("data"):
            html = base64.urlsafe_b64decode(payload["body"]["data"]).decode("utf-8", errors="replace")

        for part in payload.get("parts", []):
            t, h = self._extract_body(part)
            if t:
                text = t
            if h:
                html = h

        return text, html

    def _extract_attachments(self, payload: dict, msg_id: str,
                              attachments: list) -> None:
        if payload.get("filename") and payload.get("body", {}).get("attachmentId"):
            attachments.append({
                "filename": payload["filename"],
                "mime_type": payload.get("mimeType", ""),
                "attachment_id": payload["body"]["attachmentId"],
                "message_id": msg_id,
            })
        for part in payload.get("parts", []):
            self._extract_attachments(part, msg_id, attachments)

    def download_attachment(self, message_id: str, attachment_id: str) -> bytes:
        resp = self.service.users().messages().attachments().get(
            userId="me", messageId=message_id, id=attachment_id
        ).execute()
        return base64.urlsafe_b64decode(resp["data"])
