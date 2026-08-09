#!/usr/bin/env python3
"""Seed the running Omni stack with demo data for local development.

Creates (idempotently):
  - A demo user (demo@omni.dev / DemoPass123!) with approved KYC, an active
    PRIMARY wallet, a SAVINGS wallet, and a virtual card.
  - A counterparty user (ava@omni.dev) so transfers have a real peer.
  - ~6 months of transaction history (salary, rent, groceries, subscriptions,
    card purchases, savings transfers, interest) with a consistent running
    balance that lands exactly on the wallet balance.

Data is written through dev-only service endpoints (/api/transactions/dev/seed,
/api/wallets/dev/*), which only exist when ENVIRONMENT=local. The stack runs
with in-memory storage (MODE=memcached), so the data dies with the containers.

Usage:  make build   (builds, starts, and runs this)
        make seed    (reseed a running stack without rebuilding)

Waits for the gateway itself, so it is safe to run the moment
`docker compose up -d` returns.
"""

import json
import random
import sys
import time
import urllib.error
import urllib.request
import uuid
from datetime import datetime, timedelta, timezone

GATEWAY = "http://localhost"

DEMO_USER = {
    "email": "demo@omni.dev",
    "password": "DemoPass123!",
    "firstName": "Hugh",
    "lastName": "Scott",
    "omniTag": "HUGH1",
    "currency": "USD",
    "phone": "8761234567",
    "address": "1 Demo Way",
    "city": "Kingston",
    "state": "KGN",
    "country": "JM",
    "postalCode": "00000",
    "dob": "2002-01-01",
    "govId": "DEMO-GOV-1",
    "dataAuthorization": True,
}

PEER_USER = {
    **DEMO_USER,
    "email": "ava@omni.dev",
    "firstName": "Ava",
    "lastName": "Chen",
    "omniTag": "AVAC1",
    "govId": "DEMO-GOV-2",
}

MERCHANTS = [
    ("Blue Mountain Coffee Co", "food", 6.50, 14.00),
    ("MegaMart Kingston", "groceries", 45.00, 160.00),
    ("Island Grill", "restaurants", 12.00, 38.00),
    ("Flexy Gym", "health", 45.00, 45.00),
    ("Netflix", "entertainment", 15.49, 15.49),
    ("Spotify", "entertainment", 10.99, 10.99),
    ("Digicel Top-Up", "utilities", 20.00, 50.00),
    ("Amazon", "shopping", 18.00, 120.00),
    ("Uber", "transport", 8.00, 24.00),
    ("Fontana Pharmacy", "health", 9.00, 42.00),
]


def api(method: str, path: str, body=None) -> tuple[int, dict]:
    req = urllib.request.Request(
        GATEWAY + path,
        method=method,
        data=json.dumps(body).encode() if body is not None else None,
        headers={"Content-Type": "application/json"},
    )
    try:
        with urllib.request.urlopen(req, timeout=30) as resp:
            return resp.status, json.loads(resp.read() or b"{}")
    except urllib.error.HTTPError as e:
        try:
            return e.code, json.loads(e.read() or b"{}")
        except json.JSONDecodeError:
            return e.code, {}


def wait_for_stack(timeout: int = 90) -> None:
    """Block until the gateway can reach user-service.

    `make build` runs this right after `docker compose up -d`, so the first
    few seconds are connection-refused rather than HTTP errors — api() only
    handles the latter, so poll here before touching any real endpoint.
    """
    deadline = time.monotonic() + timeout
    waiting = False
    while time.monotonic() < deadline:
        try:
            with urllib.request.urlopen(
                GATEWAY + "/api/users/health", timeout=3
            ) as resp:
                if resp.status == 200:
                    if waiting:
                        print(" ready")
                    return
        except (urllib.error.URLError, TimeoutError, ConnectionError):
            pass  # HTTPError subclasses URLError, so non-200s keep waiting too
        if not waiting:
            print("Waiting for the stack…", end="", flush=True)
            waiting = True
        else:
            print(".", end="", flush=True)
        time.sleep(2)
    print()
    sys.exit(f"gateway never answered /api/users/health in {timeout}s — try `make logs`")


def ensure_user(payload: dict) -> dict:
    status, body = api("POST", "/api/users/auth/account/register", payload)
    if status == 201 and "user" in body:
        print(f"  registered {payload['email']} -> {body['user']['id']}")
    else:
        status, body = api(
            "POST",
            "/api/users/auth/account/login",
            {"email": payload["email"], "password": payload["password"]},
        )
        if status != 200 or "user" not in body:
            sys.exit(f"cannot register or login {payload['email']}: {status} {body}")
        print(f"  logged in {payload['email']} -> {body['user']['id']}")
    user = body["user"]
    # Complete verification the way the app does: submit registration info
    # with signed consent (auto-approves and activates wallets via Kafka).
    api("POST", f"/api/users/auth/kyc/{user['id']}/submit", {"consent": True})
    return user


def iso(dt: datetime) -> str:
    return dt.strftime("%Y-%m-%dT%H:%M:%SZ")


def build_history(
    account_id: str,
    wallet_id: str,
    savings_id: str,
    card_id: str,
    peer: dict,
    now: datetime,
) -> tuple[list[dict], float, float]:
    """Return (transactions, final_primary_balance, final_savings_balance)."""
    rng = random.Random(42)
    txs: list[dict] = []
    balance = 3200.00
    savings = 0.0

    def add(
        when: datetime,
        tx_type: str,
        category: str,
        amount: float,
        description: str,
        metadata: dict | None = None,
        card: str | None = None,
        sender=None,
        receiver=None,
        savings_leg: bool = False,
    ):
        nonlocal balance, savings
        amount = round(amount, 2)
        if savings_leg:
            before = round(savings, 2)
            savings = round(savings + amount, 2)
            after = savings
        elif category == "credit":
            before = round(balance, 2)
            balance = round(balance + amount, 2)
            after = balance
        else:
            before = round(balance, 2)
            balance = round(balance - amount, 2)
            after = balance
        txs.append(
            {
                "id": str(uuid.UUID(int=rng.getrandbits(128))),
                "reference": f"SEED-{len(txs):05d}",
                "senderAccountId": sender
                if sender is not None
                else (account_id if category == "debit" else ""),
                "receiverAccountId": receiver
                if receiver is not None
                else (account_id if category == "credit" else ""),
                "senderWalletId": wallet_id if category == "debit" else "",
                "receiverWalletId": (savings_id if savings_leg else wallet_id)
                if category == "credit"
                else "",
                **({"cardId": card} if card else {}),
                "amount": amount,
                "currency": "USD",
                "transactionType": tx_type,
                "transactionCategory": category,
                "status": "completed",
                "description": description,
                "balanceBefore": before,
                "balanceAfter": after,
                **({"metadata": metadata} if metadata else {}),
                "createdAt": iso(when),
                "completedAt": iso(when + timedelta(seconds=2)),
                "updatedAt": iso(when + timedelta(seconds=2)),
            }
        )

    start = (now - timedelta(days=182)).replace(
        hour=9, minute=0, second=0, microsecond=0
    )
    day = start
    while day < now:
        if day.day == 1:
            add(
                day.replace(hour=8),
                "deposit",
                "credit",
                4200.00,
                "Salary — Acme Corp",
                {"source": "payroll"},
            )
            add(
                day.replace(hour=10),
                "transfer",
                "debit",
                1450.00,
                "Rent — Harbour View Apts",
            )
            add(
                day.replace(hour=11),
                "transfer",
                "debit",
                500.00,
                "Auto-save to savings",
            )
            add(
                day.replace(hour=11, minute=1),
                "transfer",
                "credit",
                500.00,
                "Auto-save from primary",
                savings_leg=True,
                sender=account_id,
            )
        if day.day == 28:
            add(
                day.replace(hour=6),
                "interest_credited",
                "credit",
                max(savings * 0.003, 0.10),
                "Savings interest",
                savings_leg=True,
            )
            add(
                day.replace(hour=6, minute=5),
                "fee_charged",
                "debit",
                2.50,
                "Account service fee",
            )
        # a few card purchases most days
        for _ in range(rng.choice((0, 1, 1, 2, 2, 3))):
            name, cat, lo, hi = rng.choice(MERCHANTS)
            add(
                day.replace(hour=rng.randint(9, 21), minute=rng.randint(0, 59)),
                "card_purchase",
                "debit",
                rng.uniform(lo, hi),
                name,
                {"merchantName": name, "merchantCategory": cat},
                card=card_id,
            )
        # occasional peer transfers
        if rng.random() < 0.08:
            add(
                day.replace(hour=19),
                "transfer",
                "debit",
                rng.uniform(20, 90),
                f"Transfer to @{peer['omniTag']}",
                receiver=peer["id"],
            )
        if rng.random() < 0.06:
            add(
                day.replace(hour=20),
                "transfer",
                "credit",
                rng.uniform(25, 150),
                f"Transfer from @{peer['omniTag']}",
                sender=peer["id"],
            )
        day += timedelta(days=1)

    return txs, round(balance, 2), round(savings, 2)


def main() -> None:
    now = datetime.now(timezone.utc)

    wait_for_stack()

    print("Ensuring demo users…")
    demo = ensure_user(DEMO_USER)
    peer = ensure_user(PEER_USER)
    demo.setdefault("omniTag", DEMO_USER["omniTag"])
    peer.setdefault("omniTag", PEER_USER["omniTag"])

    # Wallet creation is async (Kafka account-created event) — wait for it.
    wallets = []
    for _ in range(15):
        status, wallets = api("GET", f"/api/wallets/list/{demo['id']}")
        if status == 200 and wallets:
            break
        time.sleep(1)
    else:
        sys.exit("demo user has no wallets after 15s — is the stack healthy?")
    primary = next((w for w in wallets if w["type"] == "PRIMARY"), wallets[0])

    status, cards = api("GET", f"/api/wallets/cards/account/{demo['id']}")
    card_id = cards["cards"][0]["id"] if status == 200 and cards.get("cards") else ""

    savings = next((w for w in wallets if w["type"] == "SAVINGS"), None)
    savings_id = savings["walletId"] if savings else str(uuid.uuid4())

    # The account's transaction list appends rather than dedupes
    # (4-transactions/src/db/implementations/memory.go:43), and every run mints
    # fresh transaction ids — so seeding twice would stack a second six months
    # of history on top of a balance that only accounts for the first.
    status, existing = api("GET", f"/api/transactions/account/{demo['id']}?limit=1")
    if status == 200 and existing:
        print("\nDemo history is already seeded — leaving it alone.")
        print("  Log in with demo@omni.dev / DemoPass123!")
        print("  `make restart` gives you a clean slate.")
        return

    print("Building 6 months of history…")
    txs, primary_balance, savings_balance = build_history(
        demo["id"], primary["walletId"], savings_id, card_id, peer, now
    )
    print(
        f"  {len(txs)} transactions, primary ${primary_balance}, savings ${savings_balance}"
    )

    print("Seeding transactions (dev endpoint)…")
    # Newest first: the in-memory store returns insertion order, so this keeps
    # `GET /account/{id}?limit=N` returning the most recent transactions.
    status, result = api("POST", "/api/transactions/dev/seed", txs[::-1])
    if status != 200:
        sys.exit(f"transaction seeding failed: {status} {result}")
    print(f"  seeded {result.get('seeded')} of {result.get('received')}")

    print("Setting wallet balances (dev endpoint)…")
    # Upsert rather than just set balance: the KYC-approved Kafka event can
    # race wallet creation, leaving the wallet inactive — force active here.
    primary.update(balance=primary_balance, status="active")
    status, result = api("POST", "/api/wallets/dev/wallets", primary)
    if status != 200:
        sys.exit(f"primary wallet update failed: {status} {result}")

    if not savings:
        savings_wallet = {
            "walletId": savings_id,
            "accountId": demo["id"],
            "type": "SAVINGS",
            "balance": savings_balance,
            "currency": "USD",
            "status": "active",
            "isDefault": False,
            "dailyLimit": 5000,
            "monthlyLimit": 20000,
            "lastActivity": iso(now),
            "createdAt": iso(now - timedelta(days=182)),
            "updatedAt": iso(now),
        }
        status, result = api("POST", "/api/wallets/dev/wallets", savings_wallet)
    else:
        status, result = api(
            "PUT",
            "/api/wallets/dev/balance",
            {"walletId": savings_id, "balance": savings_balance},
        )
    if status != 200:
        sys.exit(f"savings wallet seeding failed: {status} {result}")

    if card_id:
        # Idempotent: only top up a fresh card, re-runs shouldn't stack $500s.
        status, card = api("GET", f"/api/wallets/cards/{card_id}")
        if status == 200 and card.get("availableBalance", 0) < 500:
            api("POST", f"/api/wallets/cards/{card_id}/topup", {"amount": 500.00})

    print("\nDone. Log in with demo@omni.dev / DemoPass123!")
    print(f"  accountId: {demo['id']}")
    print(f"  primary wallet: {primary['walletId']} (${primary_balance})")
    print(f"  savings wallet: {savings_id} (${savings_balance})")
    print("\nNote: storage is in-memory, so this data dies with the containers.")
    print("`make build` and `make restart` reseed for you. `make seed` fills in a")
    print("running stack with no demo data yet; `make restart` gives a clean slate.")


if __name__ == "__main__":
    main()
