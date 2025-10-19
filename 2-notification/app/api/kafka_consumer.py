# app/api/kafka_consumer.py
import os
import json
import asyncio
from kafka import KafkaConsumer
from datetime import datetime
from typing import Dict, Any
from .models import Notification
from .database import db
from .websocket_manager import ws_manager

class NotificationKafkaConsumer:
    """Kafka consumer for notification events"""

    def __init__(self):
        self.kafka_broker = os.getenv("KAFKA_BROKER", "kafka:9092")
        self.consumer = None
        self.running = False

    def _connect(self):
        """Connect to Kafka broker with retry logic"""
        max_retries = 5
        retry_delay = 3

        for attempt in range(max_retries):
            try:
                print(f"Attempting to connect to Kafka broker at {self.kafka_broker} (attempt {attempt + 1}/{max_retries})...")
                self.consumer = KafkaConsumer(
                    "account-created",
                    "account-deletion-requested",
                    "contact-request-sent",
                    "contact-request-accepted",
                    "contact-request-rejected",
                    "contact-blocked",
                    "virtual-card-created",
                    "virtual-card-blocked",
                    "virtual-card-topped-up",
                    "physical-card-requested",
                    "virtual-card-deleted",
                    bootstrap_servers=[self.kafka_broker],
                    group_id="notification-service",
                    value_deserializer=lambda m: json.loads(m.decode('utf-8')),
                    auto_offset_reset='earliest',
                    enable_auto_commit=True
                )
                print("Successfully connected to Kafka broker!")
                return True
            except Exception as e:
                print(f"Failed to connect to Kafka (attempt {attempt + 1}/{max_retries}): {e}")
                if attempt < max_retries - 1:
                    print(f"Retrying in {retry_delay} seconds...")
                    import time
                    time.sleep(retry_delay)
                else:
                    print("Failed to connect to Kafka after all retries. Kafka consumer will not be available.")
                    return False

    async def handle_account_created(self, event: Dict[str, Any]):
        """Handle account creation event"""
        account_id = event.get("accountId")
        kyc_status = event.get("kycstatus")

        # Create welcome notification
        welcome_notification = Notification(
            account_id=account_id,
            label="Welcome to Omni!",
            content="Your account has been successfully created. Complete your KYC to activate your wallet.",
            category="account",
            type="info",
            icon="https://api.dicebear.com/7.x/initials/svg?seed=omni",
            priority="high"
        )

        # Save to database
        await db.save_notification(welcome_notification)

        # Send real-time notification via WebSocket
        await ws_manager.send_notification(account_id, welcome_notification)

        # Create wallet creation notification
        wallet_notification = Notification(
            account_id=account_id,
            label="Wallet Created",
            content=f"Your primary wallet has been created. Status: {'Active' if kyc_status == 'approved' else 'Pending KYC approval'}",
            category="wallet",
            type="success" if kyc_status == "approved" else "info",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=wallet",
            priority="normal"
        )

        await db.save_notification(wallet_notification)
        await ws_manager.send_notification(account_id, wallet_notification)

        # If KYC is pending, create KYC notification
        if kyc_status == "pending":
            kyc_notification = Notification(
                account_id=account_id,
                label="KYC Verification Pending",
                content="Please complete your KYC verification to activate full account features.",
                category="kyc",
                type="action",
                icon="https://api.dicebear.com/7.x/icons/svg?seed=kyc",
                priority="high",
                action_url="/kyc/verify"
            )

            await db.save_notification(kyc_notification)
            await ws_manager.send_notification(account_id, kyc_notification)
        elif kyc_status == "approved":
            # KYC approved notification
            kyc_notification = Notification(
                account_id=account_id,
                label="KYC Approved",
                content="Your KYC verification has been approved. You now have full access to all features!",
                category="kyc",
                type="success",
                icon="https://api.dicebear.com/7.x/icons/svg?seed=verified",
                priority="high"
            )

            await db.save_notification(kyc_notification)
            await ws_manager.send_notification(account_id, kyc_notification)

    async def handle_account_deletion(self, event: Dict[str, Any]):
        """Handle account deletion request event"""
        account_id = event.get("accountId")
        scheduled_deletion = event.get("scheduledDeletion")

        notification = Notification(
            account_id=account_id,
            label="Account Deletion Scheduled",
            content=f"Your account is scheduled for deletion on {scheduled_deletion}. You can cancel this at any time.",
            category="security",
            type="warning",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=warning",
            priority="high",
            action_url="/account/cancel-deletion"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(account_id, notification)

    async def handle_contact_request_sent(self, event: Dict[str, Any]):
        """Handle contact request sent event"""
        addressee_id = event.get("addresseeId")
        omni_Tag = event.get("omniTag")

        notification = Notification(
            account_id=addressee_id,
            label="New Contact Request",
            content=f"You received a contact request from @{omni_Tag}",
            category="contact",
            type="action",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=contact",
            priority="normal",
            action_url="/contacts/pending"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(addressee_id, notification)

    async def handle_contact_request_accepted(self, event: Dict[str, Any]):
        """Handle contact request accepted event"""
        requester_id = event.get("requesterId")

        notification = Notification(
            account_id=requester_id,
            label="Contact Request Accepted",
            content="Your contact request has been accepted!",
            category="contact",
            type="success",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=success",
            priority="normal",
            action_url="/contacts"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(requester_id, notification)

    async def handle_contact_request_rejected(self, event: Dict[str, Any]):
        """Handle contact request rejected event"""
        requester_id = event.get("requesterId")

        notification = Notification(
            account_id=requester_id,
            label="Contact Request Declined",
            content="Your contact request was declined.",
            category="contact",
            type="info",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=info",
            priority="low"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(requester_id, notification)

    async def handle_contact_blocked(self, event: Dict[str, Any]):
        """Handle contact blocked event"""
        requester_id = event.get("requesterId")
        addressee_id = event.get("addresseeId")
        blocked_by = event.get("blockedBy")

        # Notify the other person (not the blocker)
        notify_account_id = addressee_id if blocked_by == requester_id else requester_id

        notification = Notification(
            account_id=notify_account_id,
            label="Contact Unavailable",
            content="A contact is no longer available.",
            category="contact",
            type="warning",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=warning",
            priority="low"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(notify_account_id, notification)

    async def handle_virtual_card_created(self, event: Dict[str, Any]):
        """Handle virtual card created event"""
        account_id = event.get("accountId")
        last_four = event.get("lastFourDigits")
        card_type = event.get("cardType")

        notification = Notification(
            account_id=account_id,
            label="Virtual Card Created",
            content=f"Your new {card_type} card ending in {last_four} is ready to use!",
            category="card",
            type="success",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=card",
            priority="high"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(account_id, notification)

    async def handle_virtual_card_blocked(self, event: Dict[str, Any]):
        """Handle virtual card blocked event"""
        account_id = event.get("accountId")
        block_reason = event.get("blockReason")

        notification = Notification(
            account_id=account_id,
            label="Card Blocked",
            content=f"Your card has been blocked. Reason: {block_reason}",
            category="card",
            type="warning",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=blocked",
            priority="high"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(account_id, notification)

    async def handle_virtual_card_topped_up(self, event: Dict[str, Any]):
        """Handle virtual card topped up event"""
        account_id = event.get("accountId")
        amount = event.get("amount")
        new_balance = event.get("newBalance")

        notification = Notification(
            account_id=account_id,
            label="Card Topped Up",
            content=f"${amount:.2f} added to your card. New balance: ${new_balance:.2f}",
            category="card",
            type="success",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=money",
            priority="normal"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(account_id, notification)

    async def handle_physical_card_requested(self, event: Dict[str, Any]):
        """Handle physical card requested event"""
        account_id = event.get("accountId")
        delivery_city = event.get("deliveryCity")

        notification = Notification(
            account_id=account_id,
            label="Physical Card Request Received",
            content=f"Your physical card will be delivered to {delivery_city}. Processing time: 7-10 business days.",
            category="card",
            type="info",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=delivery",
            priority="normal"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(account_id, notification)

    async def handle_virtual_card_deleted(self, event: Dict[str, Any]):
        """Handle virtual card deleted event"""
        account_id = event.get("accountId")
        last_four = event.get("lastFourDigits")

        notification = Notification(
            account_id=account_id,
            label="Card Deleted",
            content=f"Your card ending in {last_four} has been permanently deleted.",
            category="card",
            type="info",
            icon="https://api.dicebear.com/7.x/icons/svg?seed=delete",
            priority="low"
        )

        await db.save_notification(notification)
        await ws_manager.send_notification(account_id, notification)

    async def process_message(self, message):
        """Process a single Kafka message"""
        topic = message.topic
        event = message.value

        try:
            if topic == "account-created":
                await self.handle_account_created(event)
            elif topic == "account-deletion-requested":
                await self.handle_account_deletion(event)
            elif topic == "contact-request-sent":
                await self.handle_contact_request_sent(event)
            elif topic == "contact-request-accepted":
                await self.handle_contact_request_accepted(event)
            elif topic == "contact-request-rejected":
                await self.handle_contact_request_rejected(event)
            elif topic == "contact-blocked":
                await self.handle_contact_blocked(event)
            elif topic == "virtual-card-created":
                await self.handle_virtual_card_created(event)
            elif topic == "virtual-card-blocked":
                await self.handle_virtual_card_blocked(event)
            elif topic == "virtual-card-topped-up":
                await self.handle_virtual_card_topped_up(event)
            elif topic == "physical-card-requested":
                await self.handle_physical_card_requested(event)
            elif topic == "virtual-card-deleted":
                await self.handle_virtual_card_deleted(event)
            else:
                print(f"Unknown topic: {topic}")

        except Exception as e:
            print(f"Error processing message from {topic}: {e}")
            import traceback
            traceback.print_exc()

    async def consume(self):
        """Main consumer loop"""
        # Try to connect to Kafka
        if not self._connect():
            print("WARNING: Kafka consumer is not available. Notification service will work without event-driven notifications.")
            return

        self.running = True
        print("Kafka consumer started. Listening for events...")

        loop = asyncio.get_event_loop()

        while self.running:
            try:
                # Run Kafka poll in executor to avoid blocking
                messages = await loop.run_in_executor(
                    None,
                    lambda: self.consumer.poll(timeout_ms=1000)
                )

                for topic_partition, records in messages.items():
                    for message in records:
                        await self.process_message(message)

                # Small sleep to prevent tight loop
                await asyncio.sleep(0.1)
            except Exception as e:
                print(f"Error in Kafka consumer loop: {e}")
                await asyncio.sleep(5)  # Wait before retrying

    def stop(self):
        """Stop the consumer"""
        self.running = False
        if self.consumer:
            self.consumer.close()
        print("Kafka consumer stopped")

# Global consumer instance
kafka_consumer = NotificationKafkaConsumer()
