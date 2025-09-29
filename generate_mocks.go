//go:generate mockgen -source=pkg/weather/client.go -destination=tests/mocks/weather_client_mock.go -package=mocks
//go:generate mockgen -source=internal/services/weather_service.go -destination=tests/mocks/weather_service_mock.go -package=mocks
//go:generate mockgen -source=internal/services/user_service.go -destination=tests/mocks/user_service_mock.go -package=mocks
//go:generate mockgen -source=internal/services/alert_service.go -destination=tests/mocks/alert_service_mock.go -package=mocks
//go:generate mockgen -source=internal/services/notification_service.go -destination=tests/mocks/notification_service_mock.go -package=mocks
//go:generate mockgen -source=internal/services/localization_service.go -destination=tests/mocks/localization_service_mock.go -package=mocks
//go:generate mockgen -source=internal/services/subscription_service.go -destination=tests/mocks/subscription_service_mock.go -package=mocks
//go:generate mockgen -source=internal/services/export_service.go -destination=tests/mocks/export_service_mock.go -package=mocks
//go:generate mockgen -source=internal/services/scheduler_service.go -destination=tests/mocks/scheduler_service_mock.go -package=mocks

package main

func main() {
	// This file is only used for generating mocks via go:generate comments
}