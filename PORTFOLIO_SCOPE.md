# Omni `v1.0-demo` portfolio scope

Status: proposed two-release finish line  
Last audited against Omni-Server and Omni-UI: 2026-07-14

## The outcome

Omni will be a polished, local-first digital-wallet demo that proves one complete story:

> A user signs in, sees live wallet and transaction data, sends money to another active Omni user, and can prove that retries and fraud rejection do not move money twice or corrupt balances.

This is a portfolio release, not a production bank or core ledger. Its value is a coherent system, honest boundaries, and executable evidence—not the number of services or features.

The release is finished when every item under **Definition of done** passes from a clean checkout and a five-minute demo can be delivered without explanation, manual data repair, or hidden setup.

The public demo is a separate follow-up release. Separating it prevents cloud work from delaying or weakening the locally proven system.

## What ships

### Core release slice

1. **Repeatable local environment**
   - `make build` starts the complete backend.
   - `make seed` creates two active counterparties plus useful dashboard history.
   - one verification command fails on any bad HTTP status or broken invariant.
   - `make down` removes the running stack cleanly.

2. **Authentication and activation**
   - seeded demo login and session restoration work.
   - consent-gated KYC changes the demo account from pending to approved.
   - wallet/card activation caused by the account events is deterministic.
   - user-owned data and money-movement endpoints reject unauthenticated callers and enforce ownership.

3. **Live customer experience**
   - Dashboard, Wallets, Transactions, and the send-money flow use backend data.
   - loading, empty, and error states are explicit; there are no silent sample-data fallbacks.
   - search, filters, and CSV export work against the loaded transaction history.

4. **Same-currency wallet transfer by OmniTag**
   - valid transfers debit and credit under one wallet-service lock.
   - the transfer conserves the combined balance and cannot overdraw the sender.
   - self-transfer, inactive wallet, missing recipient, currency mismatch, and insufficient funds are rejected.
   - replaying an idempotency key/reference returns the original result and moves no additional money.
   - an approved transfer appears once in transaction history.

5. **Rules-based fraud decision**
   - a high-risk transfer is recorded as failed with its score and reasons.
   - a rejected transfer leaves both balances unchanged.
   - the UI and documentation call this a rules engine, not machine learning.

### Supporting features: freeze, do not expand

Contacts, profile/settings, sessions, virtual cards, and real-time notifications may remain because they already demonstrate useful work. They are not allowed to expand the release. Any visible action must either work, be clearly disabled, or be removed from the release UI.

Notification claims must name the events actually consumed. Transaction notifications are outside the release unless they can be completed without delaying the core slice.

## Current audit

### Verified working

- Three of the four Go modules have passing test suites (users, wallet, transactions); fraud-detection has no tests. The existing tests cover auth handlers, KYC, wallet transfer invariants, idempotent replay, and wallet-service outage handling.
- Omni-UI's 16 Vitest tests pass.
- `make build`, `make seed`, and `make down` run successfully with Docker Compose.
- The seeded account can authenticate and load two wallets plus the seeded history (338 records observed; the count comes from a fixed-seed generator, not a defined constant).
- After activating the seeded peer manually, a live `$12.34` transfer conserved money, added one history record, and a replay moved no additional money.
- A live high-risk `$9,999.99` transfer was declined with a risk score and reasons; neither wallet balance changed.
- The UI produces and serves a Next.js production build.

### Release blockers found by the audit

1. **The default demo is not deterministic.** `make seed` left the recipient wallet inactive, so the documented transfer failed until the wallet was repaired through a dev endpoint.
2. **The smoke test is a false positive.** `make test` received two 404 responses but exited successfully and printed "Tests complete."
3. **Sensitive data reaches logs.** Registration logging included the submitted password and government ID. No credential or raw PII payload may be printed.
4. **Money movement lacks a credible public auth boundary.** The live transfer request succeeded without an authenticated cookie; the transaction API accepts a caller-supplied sender wallet ID.
5. **The frontend quality gate is bypassed.** The production build skips TypeScript and ESLint validation; standalone type checking reports many errors and lint currently fails.
6. **Some UI data is still fictional.** The Wallets page starts with faux balances, while the hidden fraud routes generate random data. This contradicts the README's no-fakes claim.
7. **Documentation overstates parts of the implementation.** Card purchases, transaction notifications, persistence, and health/smoke coverage need to be described at their real maturity level.
8. **The release Transactions page is unreachable from the nav.** The sidebar links `/transaction-history`, but the implemented route is `/transactions`; the page with search, filters, and CSV export can only be reached by typing the URL.

## Definition of done

### P0 — trustworthy demo

- [ ] `make seed` always leaves both demo users with active same-currency wallets; no manual repair is required.
- [ ] Replace the current smoke target with an assertion-based end-to-end check covering health, login/session, wallets, history, successful transfer, replay, and fraud decline.
- [ ] The check asserts balance conservation, exactly one successful history record, no movement on replay, and no movement on fraud rejection.
- [ ] A failing endpoint, 404, wrong balance, or missing record makes the command exit non-zero.
- [ ] Remove passwords, tokens, government IDs, and raw registration bodies from logs.
- [ ] Unauthenticated transfer and user-owned reads/mutations return 401/403; the authenticated account must own the sender wallet.

### P1 — honest, clean product surface

- [ ] Remove faux wallet balances and random fraud data from release routes.
- [ ] Hide or clearly disable unfinished routes and controls; do not implement AI, FX, credit score, new-account creation, physical cards, or other expansions.
- [ ] Keep only the components and route implementations used by the release; archive/delete stale alternatives that create type errors.
- [ ] `go test ./...` passes in every Go service.
- [ ] Add focused tests for fraud rules and the notification behavior still claimed by the README.
- [ ] `bun run test`, TypeScript, ESLint, and `bun run build` all pass without `ignoreBuildErrors` or `ignoreDuringBuilds`.
- [ ] The core pages work at a phone width and a desktop width without clipped primary actions.

### P2 — presentation package

- [ ] The main README states "local demo," shows start/seed/verify/UI/down commands, and lists the deliberate limitations.
- [ ] `docs/ARCHITECTURE.md` contains a system-context diagram, a transfer sequence, service/data ownership, and the trust boundary.
- [ ] `docs/DEMO.md` is a five-minute script with expected evidence and recovery instructions.
- [ ] `SECURITY.md` documents protected assets, likely threats, mitigations present, and known limitations.
- [ ] API docs and READMEs do not claim unimplemented persistence, production safety, machine learning, or transaction notifications.
- [ ] Record one short demo video or GIF and include two or three representative screenshots.
- [ ] Tag the proven commit as `v1.0-demo`.

## Five-minute demo

1. Start and seed the stack.
2. Sign in as `demo@omni.dev` and show that dashboard balances and history came from the backend.
3. Send a small amount to `@AVAC1`; show the changed sender balance and the single new history row.
4. Run the replay assertion and show that the reference and both balances remain unchanged.
5. Run the high-risk scenario and show its failed record, risk reasons, and unchanged balances.
6. Point to the architecture/security boundaries, then tear the stack down.

Registration/KYC, contacts, virtual cards, and WebSocket notifications are optional supporting moments, not extra branches in the required demo.

## Explicit non-goals for `v1.0-demo`

Do not add these before the release:

- PostgreSQL, a double-entry accounting ledger, settlement, or reconciliation
- durable distributed transactions, a transactional outbox, dead-letter infrastructure, or chaos engineering
- production persistence or migration of the unfinished Redis-backed modes
- Java/Spring, SOAP, MuleSoft, or a simulated core-banking adapter
- loans, insurance, investments, AI chat, FX trading, credit scoring, or more microservices
- real KYC providers, payment/card networks, Jam-Dex/CBDC integration, or regulatory certification
- Kubernetes, cloud production deployment, multi-region operation, or production observability
- a mobile app or a rewrite of the existing frontend
- comprehensive test coverage as a percentage target

These are reasonable future projects only after `v1.0-demo` is tagged. The README must not imply that Omni already provides them.

## The stop rule

When all P0, P1, and P2 items pass on one commit:

1. tag `v1.0-demo`;
2. record the demo;
3. freeze feature work;
4. accept only release-documentation fixes, bugs that break the demonstrated slice, or the bounded `v1.1-sandbox` work below.

New feature ideas go into a post-demo backlog. They do not move this finish line.

## `v1.1-sandbox` — hosted follow-up

Start this only after `v1.0-demo` is tagged. The goal is recruiter access, not production banking or a new architecture.

- Deploy the proven UI and Compose stack without adding services or product features.
- Expose only the HTTPS UI and API gateway; Redis, Kafka, and service ports remain private.
- Use fake demo identities, non-development secrets, rate limits, and authenticated wallet ownership checks.
- Provide a guided demo login and reset all demo data automatically on a documented schedule.
- Run the same transfer, replay, fraud-decline, and balance assertions against the public URL.
- Keep the recorded demo as a fallback if the sandbox is sleeping or unavailable.
- Choose the hosting provider and budget when this release begins; they are deliberately not `v1.0-demo` decisions.

Tag `v1.1-sandbox` when those checks pass. Then stop deployment work; production persistence, scaling, Kubernetes, and managed-service migration remain out of scope.

## Public write-up

Publish one technical case study after the sandbox is verified:

> **My Go Neobank's Smoke Test Passed Two 404s—Here's How I Made Transfers Prove Their Invariants**

Lead with the false-positive test, then show executable evidence for balance conservation, idempotent replay, fraud rejection, and the repaired auth boundary. End with the hosted demo and an explicit explanation that Omni is an educational wallet system, not a durable double-entry ledger.

This angle is intentional: useful experiments, runnable evidence, corrections, and honest limitations create a stronger hiring signal than a generic architecture tour or production-scale claims.
