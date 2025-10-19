// db/db.go

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"omni/src/models"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

var db Database

type Database interface {
	AddUser(user *models.User) error
	GetUser(email string) (*models.User, error)
	GetUserByAccountId(accountId string) (*models.User, error)
	GetUserByOmniTag(omniTag string) (*models.User, error)
	UpdateUser(user *models.User) error
	DeleteUser(email string) error
	UserExists(email string) (bool, error)
	OmniTagExists(omniTag string) (bool, error)
	AddSession(session *models.Session) error
	GetSession(id string) (*models.Session, error)
	GetUserSessions(email string) ([]*models.Session, error)
	DeleteSession(id string) error
	DeleteUserSessions(email string) error
	UpdateSessionLastLogin(id string) error
	AddRefreshToken(token string, info RefreshTokenInfo) error
	GetRefreshToken(token string) (*RefreshTokenInfo, error)
	DeleteRefreshToken(token string) error
	// Contact management methods
	SendContactRequest(requesterID, addresseeID string) (*models.Contact, error)
	AcceptContactRequest(contactID, userID string) error
	RejectContactRequest(contactID, userID string) error
	BlockContact(contactID, userID string) error
	GetContact(contactID string) (*models.Contact, error)
	GetContactsByUser(accountID string) ([]*models.ContactInfo, error)
	GetPendingRequests(accountID string) ([]*models.ContactRequest, error)
	GetSentRequests(accountID string) ([]*models.ContactRequest, error)
	DeleteContact(contactID, userID string) error
	ContactExists(requesterID, addresseeID string) (bool, error)
}

type RefreshTokenInfo struct {
	UserEmail  string
	DeviceInfo string
	CreatedAt  time.Time
}

type MemoryDB struct {
	users         map[string]models.User
	sessions      map[string]models.Session
	refreshTokens map[string]RefreshTokenInfo
	contacts      map[string]models.Contact // contactID -> Contact
	mu            sync.RWMutex
}

// SendContactRequest implements Database.
func (m *MemoryDB) SendContactRequest(requesterID string, addresseeID string) (*models.Contact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if contact already exists
	for _, contact := range m.contacts {
		if (contact.RequesterID == requesterID && contact.AddresseeID == addresseeID) ||
			(contact.RequesterID == addresseeID && contact.AddresseeID == requesterID) {
			return nil, fmt.Errorf("contact request already exists")
		}
	}

	// Create new contact
	contactID, err := generateContactID()
	if err != nil {
		return nil, err
	}

	contact := models.Contact{
		ID:          contactID,
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      models.ContactStatusPending,
		RequestedAt: time.Now(),
	}

	m.contacts[contactID] = contact
	return &contact, nil
}

// AcceptContactRequest implements Database.
func (m *MemoryDB) AcceptContactRequest(contactID string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	contact, ok := m.contacts[contactID]
	if !ok {
		return fmt.Errorf("contact not found")
	}

	// Only the addressee can accept
	if contact.AddresseeID != userID {
		return fmt.Errorf("only the addressee can accept the request")
	}

	if contact.Status != models.ContactStatusPending {
		return fmt.Errorf("contact request is not pending")
	}

	now := time.Now()
	contact.Status = models.ContactStatusAccepted
	contact.RespondedAt = &now
	m.contacts[contactID] = contact

	return nil
}

// RejectContactRequest implements Database.
func (m *MemoryDB) RejectContactRequest(contactID string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	contact, ok := m.contacts[contactID]
	if !ok {
		return fmt.Errorf("contact not found")
	}

	// Only the addressee can reject
	if contact.AddresseeID != userID {
		return fmt.Errorf("only the addressee can reject the request")
	}

	if contact.Status != models.ContactStatusPending {
		return fmt.Errorf("contact request is not pending")
	}

	now := time.Now()
	contact.Status = models.ContactStatusRejected
	contact.RespondedAt = &now
	m.contacts[contactID] = contact

	return nil
}

// BlockContact implements Database.
func (m *MemoryDB) BlockContact(contactID string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	contact, ok := m.contacts[contactID]
	if !ok {
		return fmt.Errorf("contact not found")
	}

	// Either party can block
	if contact.RequesterID != userID && contact.AddresseeID != userID {
		return fmt.Errorf("unauthorized to block this contact")
	}

	now := time.Now()
	contact.Status = models.ContactStatusBlocked
	contact.RespondedAt = &now
	m.contacts[contactID] = contact

	return nil
}

// GetContact implements Database.
func (m *MemoryDB) GetContact(contactID string) (*models.Contact, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	contact, ok := m.contacts[contactID]
	if !ok {
		return nil, fmt.Errorf("contact not found")
	}

	return &contact, nil
}

// GetContactsByUser implements Database.
func (m *MemoryDB) GetContactsByUser(accountID string) ([]*models.ContactInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var contactInfos []*models.ContactInfo

	for _, contact := range m.contacts {
		// Only show accepted contacts
		if contact.Status != models.ContactStatusAccepted {
			continue
		}

		var otherUserID string
		if contact.RequesterID == accountID {
			otherUserID = contact.AddresseeID
		} else if contact.AddresseeID == accountID {
			otherUserID = contact.RequesterID
		} else {
			continue
		}

		// Get other user's info
		otherUser, err := m.getUserByAccountIdInternal(otherUserID)
		if err != nil {
			continue
		}

		contactInfo := &models.ContactInfo{
			AccountID:  otherUser.AccountId,
			OmniTag:    otherUser.OmniTag,
			FirstName:  otherUser.FirstName,
			LastName:   otherUser.LastName,
			Email:      otherUser.Email,
			Status:     contact.Status,
			AddedAt:    *contact.RespondedAt,
			IsAccepted: true,
		}

		contactInfos = append(contactInfos, contactInfo)
	}

	return contactInfos, nil
}

// GetPendingRequests implements Database.
func (m *MemoryDB) GetPendingRequests(accountID string) ([]*models.ContactRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var requests []*models.ContactRequest

	for _, contact := range m.contacts {
		// Only show pending requests where user is addressee
		if contact.Status != models.ContactStatusPending || contact.AddresseeID != accountID {
			continue
		}

		requester, err := m.getUserByAccountIdInternal(contact.RequesterID)
		if err != nil {
			continue
		}

		request := &models.ContactRequest{
			ContactID: contact.ID,
			FromUser: models.UserBasic{
				AccountID: requester.AccountId,
				OmniTag:   requester.OmniTag,
			},
			ToUser: models.UserBasic{
				AccountID: accountID,
			},
			Status:      string(contact.Status),
			RequestedAt: contact.RequestedAt,
		}

		requests = append(requests, request)
	}

	return requests, nil
}

// GetSentRequests implements Database.
func (m *MemoryDB) GetSentRequests(accountID string) ([]*models.ContactRequest, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var requests []*models.ContactRequest

	for _, contact := range m.contacts {
		// Only show requests where user is requester
		if contact.RequesterID != accountID {
			continue
		}

		addressee, err := m.getUserByAccountIdInternal(contact.AddresseeID)
		if err != nil {
			continue
		}

		request := &models.ContactRequest{
			ContactID: contact.ID,
			FromUser: models.UserBasic{
				AccountID: accountID,
			},
			ToUser: models.UserBasic{
				AccountID: addressee.AccountId,
				OmniTag:   addressee.OmniTag,
			},
			Status:      string(contact.Status),
			RequestedAt: contact.RequestedAt,
		}

		requests = append(requests, request)
	}

	return requests, nil
}

// DeleteContact implements Database.
func (m *MemoryDB) DeleteContact(contactID string, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	contact, ok := m.contacts[contactID]
	if !ok {
		return fmt.Errorf("contact not found")
	}

	// Only parties involved can delete
	if contact.RequesterID != userID && contact.AddresseeID != userID {
		return fmt.Errorf("unauthorized to delete this contact")
	}

	delete(m.contacts, contactID)
	return nil
}

// ContactExists implements Database.
func (m *MemoryDB) ContactExists(requesterID string, addresseeID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, contact := range m.contacts {
		if (contact.RequesterID == requesterID && contact.AddresseeID == addresseeID) ||
			(contact.RequesterID == addresseeID && contact.AddresseeID == requesterID) {
			return true, nil
		}
	}
	return false, nil
}

// Helper function to get user by account ID without lock (internal use)
func (m *MemoryDB) getUserByAccountIdInternal(accountId string) (*models.User, error) {
	for _, user := range m.users {
		if user.AccountId == accountId {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

type RedisDB struct {
	client *redis.Client
}

// SendContactRequest implements Database.
func (r *RedisDB) SendContactRequest(requesterID string, addresseeID string) (*models.Contact, error) {
	ctx := context.Background()

	// Check if contact already exists
	exists, err := r.ContactExists(requesterID, addresseeID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("contact request already exists")
	}

	// Create new contact
	contactID, err := generateContactID()
	if err != nil {
		return nil, err
	}

	contact := models.Contact{
		ID:          contactID,
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      models.ContactStatusPending,
		RequestedAt: time.Now(),
	}

	contactJSON, err := json.Marshal(contact)
	if err != nil {
		return nil, err
	}

	pipe := r.client.Pipeline()
	pipe.Set(ctx, "contact:"+contactID, contactJSON, 0)
	pipe.SAdd(ctx, "user_contacts:"+requesterID, contactID)
	pipe.SAdd(ctx, "user_contacts:"+addresseeID, contactID)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return &contact, nil
}

// AcceptContactRequest implements Database.
func (r *RedisDB) AcceptContactRequest(contactID string, userID string) error {
	ctx := context.Background()

	contact, err := r.GetContact(contactID)
	if err != nil {
		return err
	}

	// Only the addressee can accept
	if contact.AddresseeID != userID {
		return fmt.Errorf("only the addressee can accept the request")
	}

	if contact.Status != models.ContactStatusPending {
		return fmt.Errorf("contact request is not pending")
	}

	now := time.Now()
	contact.Status = models.ContactStatusAccepted
	contact.RespondedAt = &now

	contactJSON, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, "contact:"+contactID, contactJSON, 0).Err()
}

// RejectContactRequest implements Database.
func (r *RedisDB) RejectContactRequest(contactID string, userID string) error {
	ctx := context.Background()

	contact, err := r.GetContact(contactID)
	if err != nil {
		return err
	}

	// Only the addressee can reject
	if contact.AddresseeID != userID {
		return fmt.Errorf("only the addressee can reject the request")
	}

	if contact.Status != models.ContactStatusPending {
		return fmt.Errorf("contact request is not pending")
	}

	now := time.Now()
	contact.Status = models.ContactStatusRejected
	contact.RespondedAt = &now

	contactJSON, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, "contact:"+contactID, contactJSON, 0).Err()
}

// BlockContact implements Database.
func (r *RedisDB) BlockContact(contactID string, userID string) error {
	ctx := context.Background()

	contact, err := r.GetContact(contactID)
	if err != nil {
		return err
	}

	// Either party can block
	if contact.RequesterID != userID && contact.AddresseeID != userID {
		return fmt.Errorf("unauthorized to block this contact")
	}

	now := time.Now()
	contact.Status = models.ContactStatusBlocked
	contact.RespondedAt = &now

	contactJSON, err := json.Marshal(contact)
	if err != nil {
		return err
	}

	return r.client.Set(ctx, "contact:"+contactID, contactJSON, 0).Err()
}

// GetContact implements Database.
func (r *RedisDB) GetContact(contactID string) (*models.Contact, error) {
	ctx := context.Background()
	contactJSON, err := r.client.Get(ctx, "contact:"+contactID).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("contact not found")
		}
		return nil, err
	}

	var contact models.Contact
	err = json.Unmarshal(contactJSON, &contact)
	if err != nil {
		return nil, err
	}

	return &contact, nil
}

// GetContactsByUser implements Database.
func (r *RedisDB) GetContactsByUser(accountID string) ([]*models.ContactInfo, error) {
	ctx := context.Background()

	contactIDs, err := r.client.SMembers(ctx, "user_contacts:"+accountID).Result()
	if err != nil {
		return nil, err
	}

	var contactInfos []*models.ContactInfo

	for _, contactID := range contactIDs {
		contact, err := r.GetContact(contactID)
		if err != nil {
			continue
		}

		// Only show accepted contacts
		if contact.Status != models.ContactStatusAccepted {
			continue
		}

		var otherUserID string
		if contact.RequesterID == accountID {
			otherUserID = contact.AddresseeID
		} else if contact.AddresseeID == accountID {
			otherUserID = contact.RequesterID
		} else {
			continue
		}

		// Get other user's info
		otherUser, err := r.GetUserByAccountId(otherUserID)
		if err != nil {
			continue
		}

		contactInfo := &models.ContactInfo{
			AccountID:  otherUser.AccountId,
			OmniTag:    otherUser.OmniTag,
			FirstName:  otherUser.FirstName,
			LastName:   otherUser.LastName,
			Email:      otherUser.Email,
			Status:     contact.Status,
			AddedAt:    *contact.RespondedAt,
			IsAccepted: true,
		}

		contactInfos = append(contactInfos, contactInfo)
	}

	return contactInfos, nil
}

// GetPendingRequests implements Database.
func (r *RedisDB) GetPendingRequests(accountID string) ([]*models.ContactRequest, error) {
	ctx := context.Background()

	contactIDs, err := r.client.SMembers(ctx, "user_contacts:"+accountID).Result()
	if err != nil {
		return nil, err
	}

	var requests []*models.ContactRequest

	for _, contactID := range contactIDs {
		contact, err := r.GetContact(contactID)
		if err != nil {
			continue
		}

		// Only show pending requests where user is addressee
		if contact.Status != models.ContactStatusPending || contact.AddresseeID != accountID {
			continue
		}

		requester, err := r.GetUserByAccountId(contact.RequesterID)
		if err != nil {
			continue
		}

		request := &models.ContactRequest{
			ContactID: contact.ID,
			FromUser: models.UserBasic{
				AccountID: requester.AccountId,
				OmniTag:   requester.OmniTag,
			},
			ToUser: models.UserBasic{
				AccountID: accountID,
			},
			Status:      string(contact.Status),
			RequestedAt: contact.RequestedAt,
		}

		requests = append(requests, request)
	}

	return requests, nil
}

// GetSentRequests implements Database.
func (r *RedisDB) GetSentRequests(accountID string) ([]*models.ContactRequest, error) {
	ctx := context.Background()

	contactIDs, err := r.client.SMembers(ctx, "user_contacts:"+accountID).Result()
	if err != nil {
		return nil, err
	}

	var requests []*models.ContactRequest

	for _, contactID := range contactIDs {
		contact, err := r.GetContact(contactID)
		if err != nil {
			continue
		}

		// Only show requests where user is requester
		if contact.RequesterID != accountID {
			continue
		}

		addressee, err := r.GetUserByAccountId(contact.AddresseeID)
		if err != nil {
			continue
		}

		request := &models.ContactRequest{
			ContactID: contact.ID,
			FromUser: models.UserBasic{
				AccountID: accountID,
			},
			ToUser: models.UserBasic{
				AccountID: addressee.AccountId,
				OmniTag:   addressee.OmniTag,
			},
			Status:      string(contact.Status),
			RequestedAt: contact.RequestedAt,
		}

		requests = append(requests, request)
	}

	return requests, nil
}

// DeleteContact implements Database.
func (r *RedisDB) DeleteContact(contactID string, userID string) error {
	ctx := context.Background()

	contact, err := r.GetContact(contactID)
	if err != nil {
		return err
	}

	// Only parties involved can delete
	if contact.RequesterID != userID && contact.AddresseeID != userID {
		return fmt.Errorf("unauthorized to delete this contact")
	}

	pipe := r.client.Pipeline()
	pipe.Del(ctx, "contact:"+contactID)
	pipe.SRem(ctx, "user_contacts:"+contact.RequesterID, contactID)
	pipe.SRem(ctx, "user_contacts:"+contact.AddresseeID, contactID)
	_, err = pipe.Exec(ctx)

	return err
}

// ContactExists implements Database.
func (r *RedisDB) ContactExists(requesterID string, addresseeID string) (bool, error) {
	ctx := context.Background()

	// Check contacts for requester
	contactIDs, err := r.client.SMembers(ctx, "user_contacts:"+requesterID).Result()
	if err != nil {
		return false, err
	}

	for _, contactID := range contactIDs {
		contact, err := r.GetContact(contactID)
		if err != nil {
			continue
		}

		if (contact.RequesterID == requesterID && contact.AddresseeID == addresseeID) ||
			(contact.RequesterID == addresseeID && contact.AddresseeID == requesterID) {
			return true, nil
		}
	}

	return false, nil
}

type FutureDB struct {
	// Placeholder for future database implementation
}

// AcceptContactRequest implements Database.
func (f *FutureDB) AcceptContactRequest(contactID string, userID string) error {
	panic("unimplemented")
}

// BlockContact implements Database.
func (f *FutureDB) BlockContact(contactID string, userID string) error {
	panic("unimplemented")
}

// ContactExists implements Database.
func (f *FutureDB) ContactExists(requesterID string, addresseeID string) (bool, error) {
	panic("unimplemented")
}

// DeleteContact implements Database.
func (f *FutureDB) DeleteContact(contactID string, userID string) error {
	panic("unimplemented")
}

// GetContact implements Database.
func (f *FutureDB) GetContact(contactID string) (*models.Contact, error) {
	panic("unimplemented")
}

// GetContactsByUser implements Database.
func (f *FutureDB) GetContactsByUser(accountID string) ([]*models.ContactInfo, error) {
	panic("unimplemented")
}

// GetPendingRequests implements Database.
func (f *FutureDB) GetPendingRequests(accountID string) ([]*models.ContactRequest, error) {
	panic("unimplemented")
}

// GetSentRequests implements Database.
func (f *FutureDB) GetSentRequests(accountID string) ([]*models.ContactRequest, error) {
	panic("unimplemented")
}

// RejectContactRequest implements Database.
func (f *FutureDB) RejectContactRequest(contactID string, userID string) error {
	panic("unimplemented")
}

// SendContactRequest implements Database.
func (f *FutureDB) SendContactRequest(requesterID string, addresseeID string) (*models.Contact, error) {
	panic("unimplemented")
}

// generateContactID creates a new UUID for contact IDs
func generateContactID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func Init() error {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	mode := strings.ToLower(os.Getenv("MODE"))

	switch {
	case env == "local" && mode == "memcached":
		db = &MemoryDB{
			users:         make(map[string]models.User),
			sessions:      make(map[string]models.Session),
			refreshTokens: make(map[string]RefreshTokenInfo),
			contacts:      make(map[string]models.Contact),
		}
	case env == "local" && mode != "memcached":
		redisPassword := os.Getenv("USER_REDIS_PASSWORD")
		redisAddr := fmt.Sprintf("user-redis:%s", os.Getenv("USER_REDIS_PORT"))
		redisClient := redis.NewClient(&redis.Options{
			Addr:     redisAddr,
			Password: redisPassword,
			DB:       0,
		})
		db = &RedisDB{client: redisClient}
		log.Println("USING DB & REDIS IN USER SERIVCE")
	case env == "local" && mode == "db":
		// Placeholder for future database implementation
		db = &FutureDB{}
	case env == "prod" || env == "production":
		// Placeholder for future database implementation
		db = &FutureDB{}
	default:
		return fmt.Errorf("unsupported environment or mode")
	}

	return nil
}

// Helper functions to call the database interface methods
func AddUser(user *models.User) error {
	exists, err := db.UserExists(user.Email)
	if err != nil {
		return fmt.Errorf("error checking user existence: %v", err)
	}
	if exists {
		return fmt.Errorf("user already exists")
	}
	return db.AddUser(user)
}

func GetUser(email string) (*models.User, error) { return db.GetUser(email) }
func GetUserByAccountId(accountId string) (*models.User, error) {
	return db.GetUserByAccountId(accountId)
}
func GetUserByOmniTag(omniTag string) (*models.User, error) {
	return db.GetUserByOmniTag(omniTag)
}
func UpdateUser(user *models.User) error                      { return db.UpdateUser(user) }
func DeleteUser(email string) error                           { return db.DeleteUser(email) }
func UserExists(email string) (bool, error)                   { return db.UserExists(email) }
func OmniTagExists(omniTag string) (bool, error)              { return db.OmniTagExists(omniTag) }
func AddSession(session *models.Session) error                { return db.AddSession(session) }
func GetSession(id string) (*models.Session, error)           { return db.GetSession(id) }
func GetUserSessions(email string) ([]*models.Session, error) { return db.GetUserSessions(email) }
func DeleteSession(id string) error                           { return db.DeleteSession(id) }
func DeleteUserSessions(email string) error                   { return db.DeleteUserSessions(email) }
func UpdateSessionLastLogin(id string) error                  { return db.UpdateSessionLastLogin(id) }
func AddRefreshToken(token string, info RefreshTokenInfo) error {
	return db.AddRefreshToken(token, info)
}
func GetRefreshToken(token string) (*RefreshTokenInfo, error) { return db.GetRefreshToken(token) }
func DeleteRefreshToken(token string) error                   { return db.DeleteRefreshToken(token) }

// Contact helper functions
func SendContactRequest(requesterID, addresseeID string) (*models.Contact, error) {
	return db.SendContactRequest(requesterID, addresseeID)
}
func AcceptContactRequest(contactID, userID string) error {
	return db.AcceptContactRequest(contactID, userID)
}
func RejectContactRequest(contactID, userID string) error {
	return db.RejectContactRequest(contactID, userID)
}
func BlockContact(contactID, userID string) error { return db.BlockContact(contactID, userID) }
func GetContact(contactID string) (*models.Contact, error) {
	return db.GetContact(contactID)
}
func GetContactsByUser(accountID string) ([]*models.ContactInfo, error) {
	return db.GetContactsByUser(accountID)
}
func GetPendingRequests(accountID string) ([]*models.ContactRequest, error) {
	return db.GetPendingRequests(accountID)
}
func GetSentRequests(accountID string) ([]*models.ContactRequest, error) {
	return db.GetSentRequests(accountID)
}
func DeleteContact(contactID, userID string) error { return db.DeleteContact(contactID, userID) }
func ContactExists(requesterID, addresseeID string) (bool, error) {
	return db.ContactExists(requesterID, addresseeID)
}

// MemoryDB implementations

func (m *MemoryDB) AddUser(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.users[user.Email] = *user
	return nil
}

func (m *MemoryDB) GetUser(email string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	user, ok := m.users[email]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return &user, nil
}
func (m *MemoryDB) GetUserByAccountId(accountId string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.users {
		if user.AccountId == accountId {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MemoryDB) GetUserByOmniTag(omniTag string) (*models.User, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.users {
		if user.OmniTag == omniTag {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (m *MemoryDB) UpdateUser(user *models.User) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[user.Email]; !ok {
		return fmt.Errorf("user not found")
	}
	m.users[user.Email] = *user
	return nil
}

func (m *MemoryDB) DeleteUser(email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.users[email]; !ok {
		return fmt.Errorf("user not found")
	}
	delete(m.users, email)
	return nil
}

func (m *MemoryDB) UserExists(email string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.users[email]
	return exists, nil
}

func (m *MemoryDB) OmniTagExists(omniTag string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, user := range m.users {
		if user.OmniTag == omniTag {
			return true, nil
		}
	}
	return false, nil
}

func (m *MemoryDB) AddSession(session *models.Session) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID] = *session
	return nil
}

func (m *MemoryDB) GetSession(id string) (*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session not found")
	}
	return &session, nil
}

func (m *MemoryDB) GetUserSessions(email string) ([]*models.Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var userSessions []*models.Session
	for _, session := range m.sessions {
		if session.UserEmail == email {
			sessionCopy := session
			userSessions = append(userSessions, &sessionCopy)
		}
	}
	return userSessions, nil
}

func (m *MemoryDB) DeleteSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.sessions[id]; !ok {
		return fmt.Errorf("session not found")
	}
	delete(m.sessions, id)
	return nil
}

func (m *MemoryDB) DeleteUserSessions(email string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, session := range m.sessions {
		if session.UserEmail == email {
			delete(m.sessions, id)
		}
	}
	return nil
}

func (m *MemoryDB) UpdateSessionLastLogin(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	session, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}
	session.LastLoginAt = time.Now()
	m.sessions[id] = session
	return nil
}

func (m *MemoryDB) AddRefreshToken(token string, info RefreshTokenInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.refreshTokens[token] = info
	return nil
}

func (m *MemoryDB) GetRefreshToken(token string) (*RefreshTokenInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, ok := m.refreshTokens[token]
	if !ok {
		return nil, fmt.Errorf("refresh token not found")
	}
	return &info, nil
}

func (m *MemoryDB) DeleteRefreshToken(token string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.refreshTokens[token]; !ok {
		return fmt.Errorf("refresh token not found")
	}
	delete(m.refreshTokens, token)
	return nil
}

// RedisDB implementations

func (r *RedisDB) AddUser(user *models.User) error {
	ctx := context.Background()
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, "user:"+user.Email, userJSON, 0).Err()
}

func (r *RedisDB) GetUser(email string) (*models.User, error) {
	ctx := context.Background()
	userJSON, err := r.client.Get(ctx, "user:"+email).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	var user models.User
	err = json.Unmarshal(userJSON, &user)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
func (r *RedisDB) GetUserByAccountId(accountId string) (*models.User, error) {
	ctx := context.Background()
	pattern := "user:*"
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		userJSON, err := r.client.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var user models.User
		if err := json.Unmarshal(userJSON, &user); err != nil {
			continue
		}
		if user.AccountId == accountId {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *RedisDB) GetUserByOmniTag(omniTag string) (*models.User, error) {
	ctx := context.Background()
	pattern := "user:*"
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		userJSON, err := r.client.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var user models.User
		if err := json.Unmarshal(userJSON, &user); err != nil {
			continue
		}
		if user.OmniTag == omniTag {
			return &user, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *RedisDB) UpdateUser(user *models.User) error {
	ctx := context.Background()
	exists, err := r.UserExists(user.Email)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("user not found")
	}
	userJSON, err := json.Marshal(user)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, "user:"+user.Email, userJSON, 0).Err()
}

func (r *RedisDB) DeleteUser(email string) error {
	ctx := context.Background()
	exists, err := r.UserExists(email)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("user not found")
	}
	return r.client.Del(ctx, "user:"+email).Err()
}

func (r *RedisDB) UserExists(email string) (bool, error) {
	ctx := context.Background()
	exists, err := r.client.Exists(ctx, "user:"+email).Result()
	if err != nil {
		return false, err
	}
	return exists == 1, nil
}

func (r *RedisDB) OmniTagExists(omniTag string) (bool, error) {
	ctx := context.Background()
	pattern := "user:*"
	iter := r.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		userJSON, err := r.client.Get(ctx, iter.Val()).Bytes()
		if err != nil {
			continue
		}
		var user models.User
		if err := json.Unmarshal(userJSON, &user); err != nil {
			continue
		}
		if user.OmniTag == omniTag {
			return true, nil
		}
	}
	return false, nil
}

func (r *RedisDB) AddSession(session *models.Session) error {
	ctx := context.Background()
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return err
	}
	pipe := r.client.Pipeline()
	pipe.Set(ctx, "session:"+session.ID, sessionJSON, 24*time.Hour)
	pipe.SAdd(ctx, "user_sessions:"+session.UserEmail, session.ID)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisDB) GetSession(id string) (*models.Session, error) {
	ctx := context.Background()
	sessionJSON, err := r.client.Get(ctx, "session:"+id).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("session not found")
		}
		return nil, err
	}
	var session models.Session
	err = json.Unmarshal(sessionJSON, &session)
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *RedisDB) GetUserSessions(email string) ([]*models.Session, error) {
	ctx := context.Background()
	sessionIDs, err := r.client.SMembers(ctx, "user_sessions:"+email).Result()
	if err != nil {
		return nil, err
	}
	var userSessions []*models.Session
	for _, id := range sessionIDs {
		session, err := r.GetSession(id)
		if err != nil {
			continue // Skip sessions that can't be retrieved
		}
		userSessions = append(userSessions, session)
	}
	return userSessions, nil
}

func (r *RedisDB) DeleteSession(id string) error {
	ctx := context.Background()
	session, err := r.GetSession(id)
	if err != nil {
		return err
	}
	pipe := r.client.Pipeline()
	pipe.Del(ctx, "session:"+id)
	pipe.SRem(ctx, "user_sessions:"+session.UserEmail, id)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisDB) DeleteUserSessions(email string) error {
	ctx := context.Background()
	sessionIDs, err := r.client.SMembers(ctx, "user_sessions:"+email).Result()
	if err != nil {
		return err
	}
	pipe := r.client.Pipeline()
	for _, id := range sessionIDs {
		pipe.Del(ctx, "session:"+id)
	}
	pipe.Del(ctx, "user_sessions:"+email)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *RedisDB) UpdateSessionLastLogin(id string) error {
	// _ := context.Background()
	session, err := r.GetSession(id)
	if err != nil {
		return err
	}
	session.LastLoginAt = time.Now()
	return r.AddSession(session) // This will overwrite the existing session
}

func (r *RedisDB) AddRefreshToken(token string, info RefreshTokenInfo) error {
	ctx := context.Background()
	infoJSON, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return r.client.Set(ctx, "refresh_token:"+token, infoJSON, 7*24*time.Hour).Err()
}

func (r *RedisDB) GetRefreshToken(token string) (*RefreshTokenInfo, error) {
	ctx := context.Background()
	infoJSON, err := r.client.Get(ctx, "refresh_token:"+token).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("refresh token not found")
		}
		return nil, err
	}
	var info RefreshTokenInfo
	err = json.Unmarshal(infoJSON, &info)
	if err != nil {
		return nil, err
	}
	return &info, nil
}

func (r *RedisDB) DeleteRefreshToken(token string) error {
	ctx := context.Background()
	return r.client.Del(ctx, "refresh_token:"+token).Err()
}

// FutureDB implementations (placeholders)

func (f *FutureDB) AddUser(user *models.User) error {
	return fmt.Errorf("FutureDB: AddUser not implemented")
}

func (f *FutureDB) GetUser(email string) (*models.User, error) {
	return nil, fmt.Errorf("FutureDB: GetUser not implemented")
}
func (f *FutureDB) GetUserByAccountId(accountId string) (*models.User, error) {
	return nil, fmt.Errorf("FutureDB: GetUserByAccountId not implemented")
}

func (f *FutureDB) GetUserByOmniTag(omniTag string) (*models.User, error) {
	return nil, fmt.Errorf("FutureDB: GetUserByOmniTag not implemented")
}

func (f *FutureDB) UpdateUser(user *models.User) error {
	return fmt.Errorf("FutureDB: UpdateUser not implemented")
}

func (f *FutureDB) DeleteUser(email string) error {
	return fmt.Errorf("FutureDB: DeleteUser not implemented")
}

func (f *FutureDB) UserExists(email string) (bool, error) {
	return false, fmt.Errorf("FutureDB: UserExists not implemented")
}

func (f *FutureDB) OmniTagExists(omniTag string) (bool, error) {
	return false, fmt.Errorf("FutureDB: OmniTagExists not implemented")
}

func (f *FutureDB) AddSession(session *models.Session) error {
	return fmt.Errorf("FutureDB: AddSession not implemented")
}

func (f *FutureDB) GetSession(id string) (*models.Session, error) {
	return nil, fmt.Errorf("FutureDB: GetSession not implemented")
}

func (f *FutureDB) GetUserSessions(email string) ([]*models.Session, error) {
	return nil, fmt.Errorf("FutureDB: GetUserSessions not implemented")
}

func (f *FutureDB) DeleteSession(id string) error {
	return fmt.Errorf("FutureDB: DeleteSession not implemented")
}

func (f *FutureDB) DeleteUserSessions(email string) error {
	return fmt.Errorf("FutureDB: DeleteUserSessions not implemented")
}

func (f *FutureDB) UpdateSessionLastLogin(id string) error {
	return fmt.Errorf("FutureDB: UpdateSessionLastLogin not implemented")
}

func (f *FutureDB) AddRefreshToken(token string, info RefreshTokenInfo) error {
	return fmt.Errorf("FutureDB: AddRefreshToken not implemented")
}

func (f *FutureDB) GetRefreshToken(token string) (*RefreshTokenInfo, error) {
	return nil, fmt.Errorf("FutureDB: GetRefreshToken not implemented")
}

func (f *FutureDB) DeleteRefreshToken(token string) error {
	return fmt.Errorf("FutureDB: DeleteRefreshToken not implemented")
}
