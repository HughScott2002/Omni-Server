# Notifications

## Account Events

### account-created
- **Recipient:** New user
- **Notifications:**
  - Welcome message
  - Wallet created status
  - KYC verification reminder (if pending) or approval (if approved)

### account-deletion-requested
- **Recipient:** User requesting deletion
- **Notification:** Account deletion scheduled with cancellation option

## Contact Events

### contact-request-sent
- **Recipient:** Addressee
- **Notification:** "You received a contact request from @{omniTag}"
- **Action:** View pending requests

### contact-request-accepted
- **Recipient:** Requester
- **Notification:** "Your contact request has been accepted!"
- **Action:** View contacts

### contact-request-rejected
- **Recipient:** Requester
- **Notification:** "Your contact request was declined."

### contact-blocked
- **Recipient:** Other party (not blocker)
- **Notification:** "A contact is no longer available."
