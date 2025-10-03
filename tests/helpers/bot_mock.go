package helpers

import (
	"strings"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/PaulSonOfLars/gotgbot/v2/ext"
)

// MockBot creates a minimal gotgbot.Bot instance for testing
type MockBot struct {
	Bot *gotgbot.Bot
}

// NewMockBot creates a new mock bot instance
func NewMockBot() *MockBot {
	return &MockBot{
		Bot: &gotgbot.Bot{
			User: gotgbot.User{
				Id:        12345,
				IsBot:     true,
				FirstName: "TestBot",
				Username:  "test_bot",
			},
			Token: "test_token",
		},
	}
}

// MockContext creates a minimal ext.Context for testing
type MockContext struct {
	Context *ext.Context
}

// MockContextOptions provides options for creating a mock context
type MockContextOptions struct {
	UserID      int64
	Username    string
	FirstName   string
	LastName    string
	ChatID      int64
	MessageID   int64
	MessageText string
	Args        []string
	CallbackID  string
	Data        string
	Latitude    float64
	Longitude   float64
}

// NewMockContext creates a new mock context with the given options
func NewMockContext(opts MockContextOptions) *MockContext {
	// Set defaults
	if opts.UserID == 0 {
		opts.UserID = 12345
	}
	if opts.Username == "" {
		opts.Username = "testuser"
	}
	if opts.FirstName == "" {
		opts.FirstName = "Test"
	}
	if opts.ChatID == 0 {
		opts.ChatID = 12345
	}
	if opts.MessageID == 0 {
		opts.MessageID = 1
	}

	message := &gotgbot.Message{
		MessageId: opts.MessageID,
		From: &gotgbot.User{
			Id:        opts.UserID,
			IsBot:     false,
			FirstName: opts.FirstName,
			LastName:  opts.LastName,
			Username:  opts.Username,
		},
		Chat: gotgbot.Chat{
			Id:   opts.ChatID,
			Type: "private",
		},
		Text: opts.MessageText,
	}

	ctx := &ext.Context{
		Update: &gotgbot.Update{
			Message: message,
		},
		EffectiveUser: &gotgbot.User{
			Id:        opts.UserID,
			IsBot:     false,
			FirstName: opts.FirstName,
			LastName:  opts.LastName,
			Username:  opts.Username,
		},
		EffectiveChat: &gotgbot.Chat{
			Id:   opts.ChatID,
			Type: "private",
		},
		EffectiveMessage: message,
	}

	// Initialize data map
	ctx.Data = make(map[string]interface{})

	// Set message text from args if provided (Args() parses from message text)
	if len(opts.Args) > 0 {
		text := strings.Join(opts.Args, " ")
		ctx.EffectiveMessage.Text = text
		ctx.Update.Message.Text = text
	}

	// Add location if coordinates provided
	if opts.Latitude != 0 || opts.Longitude != 0 {
		ctx.EffectiveMessage.Location = &gotgbot.Location{
			Latitude:  opts.Latitude,
			Longitude: opts.Longitude,
		}
	}

	// Add callback query if callback data provided
	if opts.CallbackID != "" || opts.Data != "" {
		message := &gotgbot.Message{
			MessageId: opts.MessageID,
			From: &gotgbot.User{
				Id:        opts.UserID,
				IsBot:     false,
				FirstName: opts.FirstName,
				LastName:  opts.LastName,
				Username:  opts.Username,
			},
			Chat: gotgbot.Chat{
				Id:   opts.ChatID,
				Type: "private",
			},
		}

		ctx.CallbackQuery = &gotgbot.CallbackQuery{
			Id: opts.CallbackID,
			From: gotgbot.User{
				Id:        opts.UserID,
				IsBot:     false,
				FirstName: opts.FirstName,
				LastName:  opts.LastName,
				Username:  opts.Username,
			},
			Message:      message, // Message implements MaybeInaccessibleMessage interface
			ChatInstance: "test_instance",
			Data:         opts.Data,
		}
	}

	return &MockContext{Context: ctx}
}

// NewSimpleMockContext creates a simple mock context with minimal setup
func NewSimpleMockContext(userID int64, messageText string) *MockContext {
	return NewMockContext(MockContextOptions{
		UserID:      userID,
		MessageText: messageText,
	})
}

// NewMockContextWithLocation creates a mock context with location data
func NewMockContextWithLocation(userID int64, lat, lon float64) *MockContext {
	return NewMockContext(MockContextOptions{
		UserID:    userID,
		Latitude:  lat,
		Longitude: lon,
	})
}

// NewMockContextWithCallback creates a mock context with callback query data
func NewMockContextWithCallback(userID int64, callbackID, data string) *MockContext {
	return NewMockContext(MockContextOptions{
		UserID:     userID,
		CallbackID: callbackID,
		Data:       data,
	})
}
