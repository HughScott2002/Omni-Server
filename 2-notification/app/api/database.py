# app/api/database.py
import os
import json
import redis.asyncio as redis
from typing import List, Optional
from datetime import datetime
from .models import Notification

class NotificationDatabase:
    """Redis-based notification storage"""

    def __init__(self):
        redis_host = os.getenv("REDIS_HOST", "notification-redis")
        redis_port = int(os.getenv("REDIS_PORT", "6379"))
        redis_password = os.getenv("REDIS_PASSWORD", "")

        self.redis = redis.Redis(
            host=redis_host,
            port=redis_port,
            password=redis_password,
            decode_responses=True
        )

    async def save_notification(self, notification: Notification) -> None:
        """Save a notification to Redis"""
        # Store notification by ID
        key = f"notification:{notification.notification_id}"
        await self.redis.set(key, notification.model_dump_json())

        # Add to account's notification list (sorted set by timestamp)
        account_key = f"account_notifications:{notification.account_id}"
        timestamp = notification.date.timestamp()
        await self.redis.zadd(account_key, {notification.notification_id: timestamp})

        # Track unread notifications
        if not notification.is_read:
            unread_key = f"unread_notifications:{notification.account_id}"
            await self.redis.sadd(unread_key, notification.notification_id)

    async def get_notification(self, notification_id: str) -> Optional[Notification]:
        """Get a single notification by ID"""
        key = f"notification:{notification_id}"
        data = await self.redis.get(key)
        if data:
            return Notification.model_validate_json(data)
        return None

    async def get_notifications_for_account(
        self,
        account_id: str,
        page: int = 1,
        page_size: int = 10,
        category: Optional[str] = None
    ) -> tuple[List[Notification], int]:
        """Get notifications for an account with pagination"""
        account_key = f"account_notifications:{account_id}"

        # Get total count
        total = await self.redis.zcard(account_key)

        # Calculate pagination
        start = (page - 1) * page_size
        end = start + page_size - 1

        # Get notification IDs (most recent first)
        notification_ids = await self.redis.zrevrange(account_key, start, end)

        # Fetch full notification objects
        notifications = []
        for nid in notification_ids:
            notif = await self.get_notification(nid)
            if notif:
                # Filter by category if specified
                if category is None or notif.category == category:
                    notifications.append(notif)

        return notifications, total

    async def mark_as_read(self, notification_id: str, account_id: str) -> bool:
        """Mark a notification as read"""
        notification = await self.get_notification(notification_id)
        if not notification:
            return False

        notification.is_read = True
        await self.save_notification(notification)

        # Remove from unread set
        unread_key = f"unread_notifications:{account_id}"
        await self.redis.srem(unread_key, notification_id)

        return True

    async def mark_all_as_read(self, account_id: str) -> int:
        """Mark all notifications as read for an account"""
        unread_key = f"unread_notifications:{account_id}"
        notification_ids = await self.redis.smembers(unread_key)

        count = 0
        for nid in notification_ids:
            if await self.mark_as_read(nid, account_id):
                count += 1

        return count

    async def get_unread_count(self, account_id: str) -> int:
        """Get count of unread notifications"""
        unread_key = f"unread_notifications:{account_id}"
        return await self.redis.scard(unread_key)

    async def delete_notification(self, notification_id: str, account_id: str) -> bool:
        """Delete a notification"""
        # Remove from notification storage
        key = f"notification:{notification_id}"
        result = await self.redis.delete(key)

        # Remove from account list
        account_key = f"account_notifications:{account_id}"
        await self.redis.zrem(account_key, notification_id)

        # Remove from unread set if present
        unread_key = f"unread_notifications:{account_id}"
        await self.redis.srem(unread_key, notification_id)

        return result > 0

    async def close(self):
        """Close the Redis connection"""
        await self.redis.close()

# Global database instance
db = NotificationDatabase()
