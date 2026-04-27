from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    mongo_uri: str = "mongodb://localhost:27017"
    database_name: str = "finance_agent"

    gmail_credentials_path: str = "./data/gmail_credentials.json"
    gmail_token_path: str = "./data/gmail_token.json"
    gmail_scopes: list[str] = ["https://www.googleapis.com/auth/gmail.readonly"]

    gemini_api_key: str = ""
    gemini_model: str = "gemini-2.0-flash"

    telegram_bot_token: str = ""
    telegram_chat_id: str = ""

    schedule_morning: str = "07:00"
    schedule_evening: str = "19:00"
    timezone: str = "Asia/Kolkata"

    log_level: str = "INFO"

    upload_dir: str = "./data/uploads"
    max_upload_size_mb: int = 50

    model_config = {"env_file": ".env", "env_file_encoding": "utf-8"}


settings = Settings()
