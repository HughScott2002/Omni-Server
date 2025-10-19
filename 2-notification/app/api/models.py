# app/api/models.py
from datetime import datetime
from typing import Optional, List
from pydantic import BaseModel, Field # type: ignore
import uuid

class Item(BaseModel):
    id: int
    name: str
    description: str | None = None
    
class Notification(BaseModel):
    notification_id: str = Field(default_factory=lambda: str(uuid.uuid4()))
    account_id: str
    is_read: bool = False
    was_dismissed: bool = False
    label: str
    content: str
    date: datetime = Field(default_factory=datetime.utcnow)
    type: Optional[str] = None
    icon: Optional[str] = None
    priority: Optional[str] = "normal"
    category: Optional[str] = None
    action_url: Optional[str] = None

class NotificationResponse(BaseModel):
    notifications: List[Notification]
    total: int
    page: int
    page_size: int
    unread_count: int

