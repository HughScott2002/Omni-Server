# app/api/websocket_manager.py
from fastapi import WebSocket
from typing import Dict, Set
import json
import asyncio
from .models import Notification

class WebSocketManager:
    """Manages WebSocket connections for real-time notifications"""

    def __init__(self):
        # Map of account_id -> set of WebSocket connections
        self.active_connections: Dict[str, Set[WebSocket]] = {}
        self._lock = asyncio.Lock()

    async def connect(self, websocket: WebSocket, account_id: str):
        """Accept a new WebSocket connection"""
        await websocket.accept()
        async with self._lock:
            if account_id not in self.active_connections:
                self.active_connections[account_id] = set()
            self.active_connections[account_id].add(websocket)
        print(f"Client connected: {account_id} (total: {len(self.active_connections[account_id])} connections)")

    async def disconnect(self, websocket: WebSocket, account_id: str):
        """Remove a WebSocket connection"""
        async with self._lock:
            if account_id in self.active_connections:
                self.active_connections[account_id].discard(websocket)
                if not self.active_connections[account_id]:
                    del self.active_connections[account_id]
        print(f"Client disconnected: {account_id}")

    async def send_notification(self, account_id: str, notification: Notification):
        """Send a notification to all connected clients for an account"""
        if account_id not in self.active_connections:
            print(f"No active connections for account {account_id}")
            return

        # Convert notification to JSON
        message = {
            "type": "notification",
            "data": notification.model_dump(mode="json")
        }

        # Send to all connections for this account
        disconnected = set()
        for websocket in self.active_connections[account_id]:
            try:
                await websocket.send_json(message)
            except Exception as e:
                print(f"Error sending to websocket: {e}")
                disconnected.add(websocket)

        # Clean up disconnected websockets
        if disconnected:
            async with self._lock:
                for ws in disconnected:
                    self.active_connections[account_id].discard(ws)
                if not self.active_connections[account_id]:
                    del self.active_connections[account_id]

    async def send_message(self, account_id: str, message_type: str, data: dict):
        """Send a custom message to all connected clients"""
        if account_id not in self.active_connections:
            return

        message = {
            "type": message_type,
            "data": data
        }

        disconnected = set()
        for websocket in self.active_connections[account_id]:
            try:
                await websocket.send_json(message)
            except Exception as e:
                print(f"Error sending message: {e}")
                disconnected.add(websocket)

        # Clean up disconnected websockets
        if disconnected:
            async with self._lock:
                for ws in disconnected:
                    self.active_connections[account_id].discard(ws)

    def get_connection_count(self, account_id: str) -> int:
        """Get the number of active connections for an account"""
        return len(self.active_connections.get(account_id, set()))

    def get_total_connections(self) -> int:
        """Get total number of active connections"""
        return sum(len(conns) for conns in self.active_connections.values())

# Global WebSocket manager instance
ws_manager = WebSocketManager()
