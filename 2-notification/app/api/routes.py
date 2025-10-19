# app/api/routes.py
from typing import Optional
from datetime import datetime
from fastapi import APIRouter, Query, WebSocket, WebSocketDisconnect, HTTPException
from fastapi.responses import JSONResponse
from .models import NotificationResponse, Notification
from .database import db
from .websocket_manager import ws_manager

router = APIRouter(prefix="/api/notifications")

@router.get("/health")
async def health():
    """Health check endpoint"""
    total_connections = ws_manager.get_total_connections()
    return {
        "status": "healthy",
        "service": "notification-service",
        "kafka_consumer": "running",
        "websocket_connections": total_connections
    }

@router.options("")
async def options_notifications():
    """CORS preflight handler"""
    return JSONResponse(
        content={},
        headers={
            "Access-Control-Allow-Origin": "*",
            "Access-Control-Allow-Methods": "GET, POST, OPTIONS, PUT, DELETE",
            "Access-Control-Allow-Headers": "Content-Type, Authorization",
        },
    )

@router.get("", response_model=NotificationResponse)
async def get_notifications(
    account_id: str = Query(..., description="Account ID to fetch notifications for"),
    page: int = Query(1, ge=1, description="Page number"),
    page_size: int = Query(10, ge=1, le=100, description="Number of items per page"),
    category: Optional[str] = Query(None, description="Filter by category (transaction, kyc, wallet, etc.)")
):
    """
    Get notifications for an account with pagination and optional filtering.

    - **account_id**: The user's account ID (required)
    - **page**: Page number (default: 1)
    - **page_size**: Items per page (default: 10, max: 100)
    - **category**: Optional filter by category
    """
    notifications, total = await db.get_notifications_for_account(
        account_id=account_id,
        page=page,
        page_size=page_size,
        category=category
    )

    unread_count = await db.get_unread_count(account_id)

    return NotificationResponse(
        notifications=notifications,
        total=total,
        page=page,
        page_size=page_size,
        unread_count=unread_count
    )

@router.get("/{notification_id}")
async def get_notification(notification_id: str):
    """Get a single notification by ID"""
    notification = await db.get_notification(notification_id)
    if not notification:
        raise HTTPException(status_code=404, detail="Notification not found")
    return notification

@router.put("/{notification_id}/read")
async def mark_as_read(
    notification_id: str,
    account_id: str = Query(..., description="Account ID for verification")
):
    """Mark a single notification as read"""
    success = await db.mark_as_read(notification_id, account_id)
    if not success:
        raise HTTPException(status_code=404, detail="Notification not found")

    # Send WebSocket update
    unread_count = await db.get_unread_count(account_id)
    await ws_manager.send_message(account_id, "unread_count_update", {
        "unread_count": unread_count
    })

    return {"success": True, "unread_count": unread_count}

@router.put("/read-all")
async def mark_all_as_read(account_id: str = Query(..., description="Account ID")):
    """Mark all notifications as read for an account"""
    read_count = await db.mark_all_as_read(account_id)

    # Send WebSocket update
    await ws_manager.send_message(account_id, "unread_count_update", {
        "unread_count": 0
    })

    return {"success": True, "read_count": read_count, "unread_count": 0}

@router.delete("/{notification_id}")
async def delete_notification(
    notification_id: str,
    account_id: str = Query(..., description="Account ID for verification")
):
    """Delete a notification"""
    success = await db.delete_notification(notification_id, account_id)
    if not success:
        raise HTTPException(status_code=404, detail="Notification not found")

    return {"success": True}

@router.websocket("/ws/{account_id}")
async def websocket_endpoint(websocket: WebSocket, account_id: str):
    """
    WebSocket endpoint for real-time notifications.

    Connect to this endpoint to receive notifications in real-time:
    ws://localhost:8000/api/notifications/ws/{account_id}

    Messages received:
    - type: "notification" - A new notification
    - type: "unread_count_update" - Updated unread count
    - type: "ping" - Keepalive message
    """
    await ws_manager.connect(websocket, account_id)

    try:
        # Send initial unread count
        unread_count = await db.get_unread_count(account_id)
        await websocket.send_json({
            "type": "connected",
            "data": {
                "message": "Connected to notification service",
                "account_id": account_id,
                "unread_count": unread_count
            }
        })

        # Keep connection alive and listen for messages
        while True:
            # Receive any messages from client (e.g., ping)
            data = await websocket.receive_text()

            # Echo back as pong
            if data == "ping":
                await websocket.send_json({
                    "type": "pong",
                    "data": {"timestamp": str(datetime.utcnow())}
                })

    except WebSocketDisconnect:
        await ws_manager.disconnect(websocket, account_id)
    except Exception as e:
        print(f"WebSocket error: {e}")
        await ws_manager.disconnect(websocket, account_id)

# Testing/Admin endpoints (can be removed in production)
@router.post("/test/create")
async def create_test_notification(
    account_id: str = Query(...),
    label: str = Query("Test Notification"),
    content: str = Query("This is a test notification"),
    category: str = Query("test")
):
    """Create a test notification (for testing purposes)"""
    notification = Notification(
        account_id=account_id,
        label=label,
        content=content,
        category=category,
        type="info",
        priority="normal"
    )

    await db.save_notification(notification)
    await ws_manager.send_notification(account_id, notification)

    return {"success": True, "notification": notification}
