package archive_store

import (
	"fmt"

	dbm "github.com/cosmos/cosmos-db"
)

// ArchiveStore interface provides a kvStore-like interface for archive databases
// Warning: this ArchiveStore is only usable on BeginBlock and EndBlock functions
// because it store directly the data on the disk
type ArchiveStore interface {
	// Basic operations
	Get(key []byte) []byte
	Set(key, value []byte) error
	Delete(key []byte) error
	Has(key []byte) bool

	// Iteration operations
	Iterator(start, end []byte) ArchiveIterator
	ReverseIterator(start, end []byte) ArchiveIterator

	// Batch operations
	NewBatch() ArchiveBatch

	// Utility operations
	Close() error
	Stats() map[string]string
}

// ArchiveIterator provides iteration capabilities over archive data
type ArchiveIterator interface {
	// Domain returns the domain of the iterator
	Domain() (start, end []byte)

	// Valid returns whether the current iterator is valid
	Valid() bool

	// Next moves the iterator to the next key/value pair
	Next()

	// Prev moves the iterator to the previous key/value pair
	Prev()

	// Key returns the current key
	Key() []byte

	// Value returns the current value
	Value() []byte

	// Error returns any accumulated error
	Error() error

	// Close closes the iterator
	Close() error
}

// ArchiveBatch provides batch operations for archive stores
type ArchiveBatch interface {
	// Set adds a key-value pair to the batch
	Set(key, value []byte) error

	// Delete removes a key from the batch
	Delete(key []byte) error

	// Write commits the batch
	Write() error

	// WriteSync commits the batch synchronously
	WriteSync() error

	// Close closes the batch
	Close() error

	// Get retrieves a value from the batch (if available)
	Get(key []byte) []byte

	// Has checks if a key exists in the batch
	Has(key []byte) bool
}

// ArchiveStoreType represents the type of archive store
type ArchiveStoreType string

const (
	ChronosArchiveStore ArchiveStoreType = "chronos"
	BridgeArchiveStore  ArchiveStoreType = "bridge"
)

// ArchiveStoreManager manages multiple archive stores
type ArchiveStoreManager struct {
	stores map[ArchiveStoreType]ArchiveStore
}

// NewArchiveStoreManager creates a new archive store manager
func NewArchiveStoreManager() *ArchiveStoreManager {
	return &ArchiveStoreManager{
		stores: make(map[ArchiveStoreType]ArchiveStore),
	}
}

// RegisterStore registers an archive store with the manager
func (m *ArchiveStoreManager) RegisterStore(storeType ArchiveStoreType, store ArchiveStore) {
	m.stores[storeType] = store
}

// GetStore retrieves an archive store by type
func (m *ArchiveStoreManager) GetStore(storeType ArchiveStoreType) (ArchiveStore, error) {
	store, exists := m.stores[storeType]
	if !exists {
		return nil, fmt.Errorf("archive store %s not found", storeType)
	}
	return store, nil
}

// Close closes all registered stores
func (m *ArchiveStoreManager) Close() error {
	var lastErr error
	for storeType, store := range m.stores {
		if err := store.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close %s store: %w", storeType, err)
		}
	}
	return lastErr
}

// DBArchiveStore implements ArchiveStore using a dbm.DB backend
type DBArchiveStore struct {
	db     dbm.DB
	prefix []byte
}

// NewDBArchiveStore creates a new archive store backed by dbm.DB
func NewDBArchiveStore(db dbm.DB, prefix []byte) *DBArchiveStore {
	return &DBArchiveStore{
		db:     db,
		prefix: prefix,
	}
}

// Get retrieves a value by key
func (s *DBArchiveStore) Get(key []byte) []byte {
	prefixedKey := s.prefixKey(key)
	value, _ := s.db.Get(prefixedKey)
	return value
}

// Set stores a key-value pair
func (s *DBArchiveStore) Set(key, value []byte) error {
	prefixedKey := s.prefixKey(key)
	return s.db.Set(prefixedKey, value)
}

// Delete removes a key
func (s *DBArchiveStore) Delete(key []byte) error {
	prefixedKey := s.prefixKey(key)
	return s.db.Delete(prefixedKey)
}

// Has checks if a key exists
func (s *DBArchiveStore) Has(key []byte) bool {
	prefixedKey := s.prefixKey(key)
	exists, _ := s.db.Has(prefixedKey)
	return exists
}

// Iterator creates a new iterator
func (s *DBArchiveStore) Iterator(start, end []byte) ArchiveIterator {
	prefixedStart := s.prefixKey(start)
	prefixedEnd := s.prefixKey(end)

	// If end is nil, we need to create a proper end key for the prefix
	if prefixedEnd == nil {
		prefixedEnd = s.prefixEndKey()
	}

	iter, _ := s.db.Iterator(prefixedStart, prefixedEnd)
	return &DBArchiveIterator{
		iter:   iter,
		prefix: s.prefix,
	}
}

// ReverseIterator creates a new reverse iterator
func (s *DBArchiveStore) ReverseIterator(start, end []byte) ArchiveIterator {
	prefixedStart := s.prefixKey(start)
	prefixedEnd := s.prefixKey(end)

	// If end is nil, we need to create a proper end key for the prefix
	if prefixedEnd == nil {
		prefixedEnd = s.prefixEndKey()
	}

	iter, _ := s.db.ReverseIterator(prefixedStart, prefixedEnd)
	return &DBArchiveIterator{
		iter:   iter,
		prefix: s.prefix,
	}
}

// NewBatch creates a new batch
func (s *DBArchiveStore) NewBatch() ArchiveBatch {
	return &DBArchiveBatch{
		batch:  s.db.NewBatch(),
		prefix: s.prefix,
	}
}

// Close closes the store
func (s *DBArchiveStore) Close() error {
	return s.db.Close()
}

// Stats returns database statistics
func (s *DBArchiveStore) Stats() map[string]string {
	return s.db.Stats()
}

// prefixKey adds the prefix to a key
func (s *DBArchiveStore) prefixKey(key []byte) []byte {
	if len(s.prefix) == 0 {
		return key
	}
	return append(s.prefix, key...)
}

// prefixEndKey creates an end key for the prefix
func (s *DBArchiveStore) prefixEndKey() []byte {
	if len(s.prefix) == 0 {
		return nil
	}
	endKey := make([]byte, len(s.prefix))
	copy(endKey, s.prefix)
	// Increment the last byte to create a proper end key
	for i := len(endKey) - 1; i >= 0; i-- {
		endKey[i]++
		if endKey[i] != 0 {
			break
		}
	}
	return endKey
}

// DBArchiveIterator implements ArchiveIterator using dbm.Iterator
type DBArchiveIterator struct {
	iter   dbm.Iterator
	prefix []byte
}

// Domain returns the domain of the iterator
func (i *DBArchiveIterator) Domain() (start, end []byte) {
	start, end = i.iter.Domain()
	// Remove prefix from domain keys
	if len(i.prefix) > 0 {
		if len(start) > len(i.prefix) {
			start = start[len(i.prefix):]
		}
		if len(end) > len(i.prefix) {
			end = end[len(i.prefix):]
		}
	}
	return start, end
}

// Valid returns whether the current iterator is valid
func (i *DBArchiveIterator) Valid() bool {
	return i.iter.Valid()
}

// Next moves the iterator to the next key/value pair
func (i *DBArchiveIterator) Next() {
	i.iter.Next()
}

// Prev moves the iterator to the previous key/value pair
func (i *DBArchiveIterator) Prev() {
	// Note: dbm.Iterator doesn't have Prev method, this is a limitation
	// For reverse iteration, use ReverseIterator instead
}

// Key returns the current key (without prefix)
func (i *DBArchiveIterator) Key() []byte {
	key := i.iter.Key()
	if len(i.prefix) > 0 && len(key) > len(i.prefix) {
		return key[len(i.prefix):]
	}
	return key
}

// Value returns the current value
func (i *DBArchiveIterator) Value() []byte {
	return i.iter.Value()
}

// Error returns any accumulated error
func (i *DBArchiveIterator) Error() error {
	return i.iter.Error()
}

// Close closes the iterator
func (i *DBArchiveIterator) Close() error {
	return i.iter.Close()
}

// DBArchiveBatch implements ArchiveBatch using dbm.Batch
type DBArchiveBatch struct {
	batch  dbm.Batch
	prefix []byte
	cache  map[string][]byte // For Get/Has operations on uncommitted data
}

// Set adds a key-value pair to the batch
func (b *DBArchiveBatch) Set(key, value []byte) error {
	prefixedKey := b.prefixKey(key)
	if b.cache == nil {
		b.cache = make(map[string][]byte)
	}
	b.cache[string(key)] = value
	return b.batch.Set(prefixedKey, value)
}

// Delete removes a key from the batch
func (b *DBArchiveBatch) Delete(key []byte) error {
	prefixedKey := b.prefixKey(key)
	if b.cache == nil {
		b.cache = make(map[string][]byte)
	}
	delete(b.cache, string(key))
	return b.batch.Delete(prefixedKey)
}

// Write commits the batch
func (b *DBArchiveBatch) Write() error {
	return b.batch.Write()
}

// WriteSync commits the batch synchronously
func (b *DBArchiveBatch) WriteSync() error {
	return b.batch.WriteSync()
}

// Close closes the batch
func (b *DBArchiveBatch) Close() error {
	return b.batch.Close()
}

// Get retrieves a value from the batch (if available)
func (b *DBArchiveBatch) Get(key []byte) []byte {
	if b.cache != nil {
		if value, exists := b.cache[string(key)]; exists {
			return value
		}
	}
	return nil
}

// Has checks if a key exists in the batch
func (b *DBArchiveBatch) Has(key []byte) bool {
	if b.cache != nil {
		_, exists := b.cache[string(key)]
		return exists
	}
	return false
}

// prefixKey adds the prefix to a key
func (b *DBArchiveBatch) prefixKey(key []byte) []byte {
	if len(b.prefix) == 0 {
		return key
	}
	return append(b.prefix, key...)
}

// ArchiveStoreUtils provides utility functions for working with archive stores
type ArchiveStoreUtils struct {
	manager *ArchiveStoreManager
}

// NewArchiveStoreUtils creates new archive store utilities
func NewArchiveStoreUtils(manager *ArchiveStoreManager) *ArchiveStoreUtils {
	return &ArchiveStoreUtils{
		manager: manager,
	}
}

// GetAllKeys retrieves all keys from a store
func (u *ArchiveStoreUtils) GetAllKeys(storeType ArchiveStoreType) ([][]byte, error) {
	store, err := u.manager.GetStore(storeType)
	if err != nil {
		return nil, err
	}

	var keys [][]byte
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		keys = append(keys, key)
	}

	return keys, iter.Error()
}

// GetAllKeyValuePairs retrieves all key-value pairs from a store
func (u *ArchiveStoreUtils) GetAllKeyValuePairs(storeType ArchiveStoreType) (map[string][]byte, error) {
	store, err := u.manager.GetStore(storeType)
	if err != nil {
		return nil, err
	}

	pairs := make(map[string][]byte)
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := string(iter.Key())
		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())
		pairs[key] = value
	}

	return pairs, iter.Error()
}

// GetKeysByPrefix retrieves all keys with a specific prefix
func (u *ArchiveStoreUtils) GetKeysByPrefix(storeType ArchiveStoreType, prefix []byte) ([][]byte, error) {
	store, err := u.manager.GetStore(storeType)
	if err != nil {
		return nil, err
	}

	var keys [][]byte
	iter := store.Iterator(prefix, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := make([]byte, len(iter.Key()))
		copy(key, iter.Key())
		keys = append(keys, key)
	}

	return keys, iter.Error()
}

// GetKeyValuePairsByPrefix retrieves all key-value pairs with a specific prefix
func (u *ArchiveStoreUtils) GetKeyValuePairsByPrefix(storeType ArchiveStoreType, prefix []byte) (map[string][]byte, error) {
	store, err := u.manager.GetStore(storeType)
	if err != nil {
		return nil, err
	}

	pairs := make(map[string][]byte)
	iter := store.Iterator(prefix, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := string(iter.Key())
		value := make([]byte, len(iter.Value()))
		copy(value, iter.Value())
		pairs[key] = value
	}

	return pairs, iter.Error()
}

// CountKeys counts the total number of keys in a store
func (u *ArchiveStoreUtils) CountKeys(storeType ArchiveStoreType) (int, error) {
	store, err := u.manager.GetStore(storeType)
	if err != nil {
		return 0, err
	}

	count := 0
	iter := store.Iterator(nil, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		count++
	}

	return count, iter.Error()
}

// DeleteByPrefix deletes all keys with a specific prefix
func (u *ArchiveStoreUtils) DeleteByPrefix(storeType ArchiveStoreType, prefix []byte) error {
	store, err := u.manager.GetStore(storeType)
	if err != nil {
		return err
	}

	batch := store.NewBatch()
	defer batch.Close()

	iter := store.Iterator(prefix, nil)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		if err := batch.Delete(iter.Key()); err != nil {
			return err
		}
	}

	return batch.Write()
}

// CompactStore compacts a store (if supported by the underlying database)
func (u *ArchiveStoreUtils) CompactStore(storeType ArchiveStoreType) error {
	store, err := u.manager.GetStore(storeType)
	if err != nil {
		return err
	}

	// Try to cast to a store that supports compaction
	if compactable, ok := store.(interface{ Compact() error }); ok {
		return compactable.Compact()
	}

	return fmt.Errorf("store %s does not support compaction", storeType)
}
