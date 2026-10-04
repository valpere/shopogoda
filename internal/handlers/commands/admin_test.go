package commands

import (
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/pkg/metrics"
	"github.com/valpere/shopogoda/tests/helpers"
)

// Helper function to create services for testing
func newTestServices(mockDB *helpers.MockDB) *services.Services {
	logger := zerolog.Nop()
	metricsCollector := metrics.New()
	startTime := time.Now()

	userService := services.NewUserService(mockDB.DB, metricsCollector, &logger, startTime)
	localizationService := services.NewLocalizationService(&logger)

	return &services.Services{
		User:         userService,
		Localization: localizationService,
	}
}

func TestCommandHandler_Promote(t *testing.T) {
	t.Run("successful promotion with usage help", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)

		// Mock admin user
		adminRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		adminRows.AddRow(
			adminID, "admin", "Admin", "User", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(adminID).
			WillReturnRows(adminRows)

		// Create context with only command (no args)
		mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
			UserID: adminID,
			Args:   []string{"/promote"}, // No user_id provided
		})

		err := handler.Promote(mockBot, mockCtx.Context)

		// Should send usage message
		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("non-admin cannot promote", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		userID := int64(200)

		// Mock regular user (not admin)
		userRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		userRows.AddRow(
			userID, "user", "Regular", "User", "en-US",
			true, models.RoleUser, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(userID).
			WillReturnRows(userRows)

		mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
			UserID: userID,
			Args:   []string{"/promote", "123"},
		})

		err := handler.Promote(mockBot, mockCtx.Context)

		// Should fail with insufficient permissions
		assert.NoError(t, err) // Command completes, but sends error message to user
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("promote user to moderator with confirmation", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)
		targetUserID := int64(200)

		// User cache miss for admin user (getUserLanguage call)

		// Mock admin user query
		adminRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		adminRows.AddRow(
			adminID, "admin", "Admin", "User", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(adminID).
			WillReturnRows(adminRows)

		// User cache HIT for admin user (permission check reuses cached value from previous call)
		// Note: In real execution, SetEx would cache after the first GetUser, but we skip mocking SetEx
		// and directly mock the cache hit since we're testing command logic, not caching behavior

		// User cache miss for target user

		// Mock target user query
		targetRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		targetRows.AddRow(
			targetUserID, "target", "Target", "User", "en-US",
			true, models.RoleUser, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(targetUserID).
			WillReturnRows(targetRows)

		mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
			UserID: adminID,
			Args:   []string{"/promote", "200"},
		})

		err := handler.Promote(mockBot, mockCtx.Context)

		// Should send confirmation dialog
		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
		// Note: We don't verify cache state here because we're not fully mocking
		// the caching behavior (SetEx calls). The test focuses on command logic.
	})

	t.Run("invalid user ID format", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)

		// Mock admin user
		adminRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		adminRows.AddRow(
			adminID, "admin", "Admin", "User", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(adminID).
			WillReturnRows(adminRows)

		mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
			UserID: adminID,
			Args:   []string{"/promote", "invalid"},
		})

		err := handler.Promote(mockBot, mockCtx.Context)

		// Should send invalid user ID error
		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("invalid role argument", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)
		targetUserID := int64(200)

		// User cache miss for admin user (getUserLanguage call)

		// Mock admin user query
		adminRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		adminRows.AddRow(
			adminID, "admin", "Admin", "User", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(adminID).
			WillReturnRows(adminRows)

		// User cache HIT for admin user (permission check reuses cached value)

		// User cache miss for target user

		// Mock target user query
		targetRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		targetRows.AddRow(
			targetUserID, "target", "Target", "User", "en-US",
			true, models.RoleUser, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(targetUserID).
			WillReturnRows(targetRows)

		mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
			UserID: adminID,
			Args:   []string{"/promote", "200", "invalid_role"},
		})

		err := handler.Promote(mockBot, mockCtx.Context)

		// Should send invalid role error
		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
		// Note: We don't verify cache state here because we're not fully mocking
		// the caching behavior (SetEx calls). The test focuses on command logic.
	})
}

func TestCommandHandler_Demote(t *testing.T) {
	t.Run("successful demotion with usage help", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)

		// Mock admin user
		adminRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		adminRows.AddRow(
			adminID, "admin", "Admin", "User", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(adminID).
			WillReturnRows(adminRows)

		// Create context with only command (no args)
		mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
			UserID: adminID,
			Args:   []string{"/demote"}, // No user_id provided
		})

		err := handler.Demote(mockBot, mockCtx.Context)

		// Should send usage message
		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("non-admin cannot demote", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		userID := int64(200)

		// Mock regular user (not admin)
		userRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		userRows.AddRow(
			userID, "user", "Regular", "User", "en-US",
			true, models.RoleUser, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(userID).
			WillReturnRows(userRows)

		mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
			UserID: userID,
			Args:   []string{"/demote", "123"},
		})

		err := handler.Demote(mockBot, mockCtx.Context)

		// Should fail with insufficient permissions
		assert.NoError(t, err) // Command completes, but sends error message to user
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("demote admin to moderator with warning", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)
		targetAdminID := int64(200)

		// User cache miss for admin user (getUserLanguage call)

		// Mock admin user query
		adminRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		adminRows.AddRow(
			adminID, "admin", "Admin", "User", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(adminID).
			WillReturnRows(adminRows)

		// User cache HIT for admin user (permission check reuses cached value)

		// User cache miss for target admin

		// Mock target admin query
		targetRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		targetRows.AddRow(
			targetAdminID, "target_admin", "Target", "Admin", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(targetAdminID).
			WillReturnRows(targetRows)

		mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
			UserID: adminID,
			Args:   []string{"/demote", "200"},
		})

		err := handler.Demote(mockBot, mockCtx.Context)

		// Should send confirmation dialog with warning
		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("cannot demote user role", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)
		targetUserID := int64(200)

		// User cache miss for admin user (getUserLanguage call)

		// Mock admin user query
		adminRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		adminRows.AddRow(
			adminID, "admin", "Admin", "User", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(adminID).
			WillReturnRows(adminRows)

		// User cache HIT for admin user (permission check reuses cached value)

		// User cache miss for target user

		// Mock target user query (already lowest role)
		targetRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		targetRows.AddRow(
			targetUserID, "target", "Target", "User", "en-US",
			true, models.RoleUser, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(targetUserID).
			WillReturnRows(targetRows)

		mockCtx := helpers.NewMockContext(helpers.MockContextOptions{
			UserID: adminID,
			Args:   []string{"/demote", "200"},
		})

		err := handler.Demote(mockBot, mockCtx.Context)

		// Should send "already lowest role" message
		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestCommandHandler_confirmRoleChange(t *testing.T) {
	t.Run("successful role change confirmation", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)
		targetUserID := int64(200)

		// User cache miss for target user (confirmRoleChange calls GetUser for target)

		// Mock target user query (before change)
		targetRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		targetRows.AddRow(
			targetUserID, "target", "Target", "User", "en-US",
			true, models.RoleUser, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(targetUserID).
			WillReturnRows(targetRows)

		// ChangeUserRole calls GetUser for admin first
		// User cache miss for admin user

		// Mock admin user query
		adminRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		adminRows.AddRow(
			adminID, "admin", "Admin", "User", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(adminID).
			WillReturnRows(adminRows)

		// ChangeUserRole calls GetUser for target user again
		// User cache HIT for target user (already cached from first GetUser)

		// Note: Admin count query is NOT needed here because we're promoting User→Moderator (not demoting an admin)

		// Mock cache invalidation

		// Mock role update (GORM automatically wraps Update in a transaction)
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "users" SET "role"=\$1,"updated_at"=\$2 WHERE id = \$3`).
			WithArgs(models.RoleModerator, helpers.AnyTime{}, targetUserID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		mockCtx := helpers.NewMockContextWithCallback(adminID, "test_callback", "role_confirm_promote_200_2")

		params := []string{"200", "2"} // targetUserID, newRole (Moderator)
		err := handler.confirmRoleChange(mockBot, mockCtx.Context, params)

		require.NoError(t, err)

		// Check mock expectations - this will show what queries were NOT executed
		err = mockDB.Mock.ExpectationsWereMet()
		if err != nil {
			t.Logf("Unmatched expectations: %v", err)
		}
		mockDB.ExpectationsWereMet(t)
		// Note: We don't verify cache state here because we're not fully mocking
		// the caching behavior (SetEx calls). The test focuses on command logic.
	})

	t.Run("invalid callback params", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)

		mockCtx := helpers.NewMockContextWithCallback(adminID, "test_callback", "role_confirm_promote")

		params := []string{} // Empty params
		err := handler.confirmRoleChange(mockBot, mockCtx.Context, params)

		// Should send error message
		assert.NoError(t, err)
	})
}

func TestCommandHandler_cancelRoleChange(t *testing.T) {
	t.Run("successful role change cancellation", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)

		mockCtx := helpers.NewMockContextWithCallback(adminID, "test_callback", "role_cancel")

		err := handler.cancelRoleChange(mockBot, mockCtx.Context)

		// Should edit message with cancellation text
		assert.NoError(t, err)
	})
}

func TestCommandHandler_handleRoleCallback(t *testing.T) {
	t.Run("route to confirm action", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)
		targetUserID := int64(200)

		// User cache miss for target user (confirmRoleChange calls GetUser for target)

		// Mock target user query
		targetRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		targetRows.AddRow(
			targetUserID, "target", "Target", "User", "en-US",
			true, models.RoleUser, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(targetUserID).
			WillReturnRows(targetRows)

		// ChangeUserRole calls GetUser for admin first
		// User cache miss for admin user

		// Mock admin user query
		adminRows := mockDB.Mock.NewRows([]string{
			"id", "username", "first_name", "last_name", "language",
			"is_active", "role", "created_at", "updated_at",
		})
		adminRows.AddRow(
			adminID, "admin", "Admin", "User", "en-US",
			true, models.RoleAdmin, time.Now(), time.Now(),
		)
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users"`).
			WithArgs(adminID).
			WillReturnRows(adminRows)

		// ChangeUserRole calls GetUser for target user again
		// User cache HIT for target user (already cached from first GetUser)

		// Note: Admin count query is NOT needed here because we're promoting User→Moderator (not demoting an admin)

		// Mock cache invalidation

		// Mock role update (GORM automatically wraps Update in a transaction)
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "users" SET "role"=\$1,"updated_at"=\$2 WHERE id = \$3`).
			WithArgs(models.RoleModerator, helpers.AnyTime{}, targetUserID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		mockCtx := helpers.NewMockContextWithCallback(adminID, "test_callback", "role_confirm_promote_200_2")

		params := []string{"200", "2"}
		err := handler.handleRoleCallback(mockBot, mockCtx.Context, "confirm", params)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
		// Note: We don't verify cache state here because we're not fully mocking
		// the caching behavior (SetEx calls). The test focuses on command logic.
	})

	t.Run("route to cancel action", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)

		mockCtx := helpers.NewMockContextWithCallback(adminID, "test_callback", "role_cancel")

		params := []string{}
		err := handler.handleRoleCallback(mockBot, mockCtx.Context, "cancel", params)

		assert.NoError(t, err)
	})

	t.Run("unknown action", func(t *testing.T) {
		mockDB := helpers.NewMockDB(t)
		defer func() { _ = mockDB.Close() }()
		logger := zerolog.Nop()
		testServices := newTestServices(mockDB)
		handler := New(testServices, &logger)

		mockBot := helpers.NewMockBot().Bot
		adminID := int64(100)

		mockCtx := helpers.NewMockContextWithCallback(adminID, "test_callback", "role_unknown_action")

		params := []string{}
		err := handler.handleRoleCallback(mockBot, mockCtx.Context, "unknown", params)

		// Should log warning and return nil
		assert.NoError(t, err)
	})
}
