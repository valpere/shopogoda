// Package tests contains test utilities and mock generation directives.
//
// NOTE: This project uses concrete types for services rather than interfaces,
// so gomock/mockgen is not currently used.
//
// For testing, the project uses:
// - a real SQLite file in t.TempDir() for integration tests (no Docker needed)
// - sqlmock for database mocking in unit tests
// - Custom bot mocks in tests/helpers/bot_mock.go
//
// If you need to add interface-based mocking in the future:
// 1. Define interfaces in internal/interfaces/
// 2. Add go:generate directives below
// 3. Run: make generate-mocks
//
// Example (currently disabled):
// //go:generate mockgen -source=../internal/interfaces/services.go -destination=mocks/services_mock.go -package=mocks

package tests
