"""Finance Agent entry point -- FastAPI app + scheduler + Telegram bot."""

import asyncio
import logging
import os
import sys

import uvicorn
from fastapi import FastAPI

from config.settings import settings
from src.db.mongo import ensure_indexes, close_connection
from src.scheduler import setup_schedule, get_scheduler, run_sync_pipeline

logging.basicConfig(
    level=getattr(logging, settings.log_level.upper(), logging.INFO),
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
    handlers=[
        logging.StreamHandler(sys.stdout),
        logging.FileHandler("logs/finance_agent.log", mode="a"),
    ],
)
logger = logging.getLogger(__name__)

app = FastAPI(
    title="Finance Agent",
    description="Personal finance tracking agent (CRED Money style)",
    version="1.0.0",
)


@app.on_event("startup")
async def startup():
    os.makedirs("logs", exist_ok=True)
    os.makedirs(settings.upload_dir, exist_ok=True)

    logger.info("Starting Finance Agent...")

    ensure_indexes()
    logger.info("MongoDB indexes ready")

    setup_schedule()
    scheduler = get_scheduler()
    scheduler.start()
    logger.info("Scheduler started")

    # Start Telegram bot polling in background
    if settings.telegram_bot_token:
        try:
            from src.reports.telegram_bot import get_bot_app
            bot_app = get_bot_app()
            await bot_app.initialize()
            await bot_app.start()
            await bot_app.updater.start_polling(drop_pending_updates=True)
            logger.info("Telegram bot started")
        except Exception:
            logger.exception("Failed to start Telegram bot")
    else:
        logger.warning("Telegram bot token not configured, skipping bot startup")

    # Run initial sync
    logger.info("Running initial sync...")
    try:
        count = await run_sync_pipeline()
        logger.info("Initial sync complete: %d transactions", count)
    except Exception:
        logger.exception("Initial sync failed (will retry on schedule)")


@app.on_event("shutdown")
async def shutdown():
    logger.info("Shutting down Finance Agent...")

    scheduler = get_scheduler()
    if scheduler.running:
        scheduler.shutdown(wait=False)

    if settings.telegram_bot_token:
        try:
            from src.reports.telegram_bot import get_bot_app
            bot_app = get_bot_app()
            await bot_app.updater.stop()
            await bot_app.stop()
            await bot_app.shutdown()
        except Exception:
            pass

    close_connection()
    logger.info("Shutdown complete")


@app.get("/health")
async def health():
    return {"status": "ok", "service": "finance-agent"}


# Import API routes
from src.api.routes import router as api_router
app.include_router(api_router, prefix="/api/v1")


if __name__ == "__main__":
    uvicorn.run(
        "src.main:app",
        host="0.0.0.0",
        port=8000,
        reload=False,
        log_level=settings.log_level.lower(),
    )
