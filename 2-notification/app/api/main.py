# app/api/main.py
import asyncio
from contextlib import asynccontextmanager
from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware
from .routes import router
from .kafka_consumer import kafka_consumer
from .database import db

# Background task for Kafka consumer
kafka_task = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    """Manage application lifecycle"""
    global kafka_task

    # Startup
    print("Starting notification service...")

    # Start Kafka consumer in background
    kafka_task = asyncio.create_task(kafka_consumer.consume())
    print("Kafka consumer task started")

    yield

    # Shutdown
    print("Shutting down notification service...")
    kafka_consumer.stop()

    if kafka_task:
        kafka_task.cancel()
        try:
            await kafka_task
        except asyncio.CancelledError:
            pass

    await db.close()
    print("Notification service stopped")

app = FastAPI(
    title="Notification Service",
    description="Real-time notification service with WebSocket support",
    version="2.0.0",
    lifespan=lifespan
)

# Add CORS middleware to allow cross-origin requests
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # In production, specify your frontend domain
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

app.include_router(router)

@app.get("/")
async def root():
    return {
        "service": "Omni Notification Service",
        "version": "2.0.0",
        "endpoints": {
            "health": "/api/notifications/health",
            "docs": "/docs",
            "websocket": "ws://<host>/api/notifications/ws/{account_id}"
        }
    }
