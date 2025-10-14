# Demo Mode Guide

ShoPogoda includes a comprehensive demo mode for testing, presentations, and evaluation purposes. Demo mode automatically populates the database with realistic demonstration data.

## Quick Start

### Enable Demo Mode

Add to your `.env` file:

```bash
DEMO_MODE=true
```

When the bot starts with demo mode enabled, it automatically creates:

- ✅ **Demo User** (ID: 999999999)
- ✅ **24 hours of weather data** with realistic patterns
- ✅ **3 alert configurations** (temperature, humidity, air quality)
- ✅ **3 notification subscriptions** (daily, weekly, alerts)

### Start the Bot

```bash
make run
```

You'll see in the logs:

```json
{"level":"info","message":"Demo mode enabled - seeding demo data"}
{"level":"info","message":"Demo user created","user_id":999999999}
{"level":"info","message":"Demo weather data created","count":24}
{"level":"info","message":"Demo alerts created","count":3}
{"level":"info":"Demo subscriptions created","count":3}
{"level":"info","message":"Demo data seeded successfully"}
```

## Demo User Details

| Property | Value |
|----------|-------|
| **Telegram ID** | 999999999 |
| **Username** | demo_user |
| **Name** | Demo User |
| **Location** | Kyiv, Ukraine (50.4501°N, 30.5234°E) |
| **Timezone** | Europe/Kyiv |
| **Language** | English |
| **Units** | Metric |
| **Role** | User |

## Seeded Data

### Weather Data (24 hours)

- **Records**: Hourly weather data for the last 24 hours
- **Temperature**: Realistic variation (10-25°C with sine wave pattern)
- **Conditions**: Automatically varied (Freezing, Cold, Cool, Mild, Warm, Hot)
- **Wind**: 5-13 km/h with varying direction
- **Humidity**: 60-80%
- **Pressure**: 1013-1018 hPa
- **Air Quality**: Last 6 hours include AQI data (50-80 range)

### Alert Configurations (3 alerts)

1. **Temperature Alert**
   - Type: Temperature
   - Condition: Greater than
   - Threshold: 30.0°C
   - Status: Active ✅

2. **Humidity Alert**
   - Type: Humidity
   - Condition: Greater than
   - Threshold: 80.0%
   - Status: Active ✅

3. **Air Quality Alert**
   - Type: Air Quality (AQI)
   - Condition: Greater than
   - Threshold: 100
   - Status: Inactive ⏸️

### Notification Subscriptions (3 subscriptions)

1. **Daily Weather Update**
   - Type: Daily
   - Frequency: Daily
   - Delivery Time: 08:00
   - Status: Active ✅

2. **Weekly Forecast**
   - Type: Weekly
   - Frequency: Weekly
   - Delivery Time: 09:00
   - Status: Active ✅

3. **Alert Notifications**
   - Type: Alerts
   - Frequency: Hourly
   - Status: Active ✅

## Testing with Demo Data

### View Demo Data in Bot

You can query the demo data using standard bot commands:

```plaintext
/weather        → Current weather for Kyiv
/forecast       → 5-day forecast
/air            → Air quality information
/alerts         → View configured alerts
/subscriptions  → View notification subscriptions
/settings       → Access all settings
```

### Export Demo Data

Test the export functionality:

```plaintext
/settings → 📊 Data Export → All Data → JSON
```

This generates a complete export of all demo user data.

## Admin Commands

Demo mode includes admin commands for managing demonstration data:

### Reset Demo Data

```plaintext
/demoreset
```

**Admin only** - Clears existing demo data and re-seeds with fresh data.

Use this when:

- Demo data becomes outdated
- Testing data migrations
- Preparing for presentations
- Resetting after testing

### Clear Demo Data

```plaintext
/democlear
```

**Admin only** - Removes all demo data from the database.

Use this when:

- Disabling demo mode
- Cleaning up test environment
- Preparing for production deployment

## Use Cases

### 1. Development & Testing

Enable demo mode during development to have consistent test data:

```bash
DEMO_MODE=true make dev
```

Benefits:

- Immediate data availability
- Consistent test scenarios
- No manual data entry
- Faster iteration

### 2. Demonstrations & Presentations

Showcase bot features with realistic data:

1. Enable demo mode
2. Start bot
3. Show /weather, /forecast, /air commands
4. Display configured alerts and subscriptions
5. Demonstrate export functionality

### 3. Integration Testing

Use demo data for automated testing:

```go
// In tests
if cfg.Bot.DemoMode {
    // Test against known demo user
    userID := services.DemoUserID
    // Run test scenarios
}
```

### 4. Documentation Screenshots

Generate consistent screenshots for documentation:

1. Enable demo mode
2. Interact with demo user
3. Capture screenshots
4. Reset with `/demoreset` for fresh state

## Architecture

### Demo Service

Location: `internal/services/demo_service.go`

```go
type DemoService struct {
    db     *gorm.DB
    logger *zerolog.Logger
}

// Key methods
func (s *DemoService) SeedDemoData(ctx context.Context) error
func (s *DemoService) ClearDemoData(ctx context.Context) error
func (s *DemoService) ResetDemoData(ctx context.Context) error
func (s *DemoService) IsDemoUser(userID int64) bool
```

### Data Generation

Weather data uses realistic patterns:

```go
// Temperature: Sine wave for natural daily variation
baseTemp := 15.0
variation := 8.0
temperature := baseTemp + variation*0.5*(1+0.8*((hour-6)/12))

// Conditions: Based on temperature
func getWeatherDescription(temp int) string {
    switch {
    case temp < 0:  return "Freezing"
    case temp < 10: return "Cold"
    case temp < 20: return "Cool"
    case temp < 25: return "Mild"
    case temp < 30: return "Warm"
    default:        return "Hot"
    }
}
```

## Configuration

### Environment Variable

```bash
DEMO_MODE=true     # Enable demo mode
DEMO_MODE=false    # Disable demo mode (default)
```

### YAML Configuration

```yaml
bot:
  demo_mode: true
```

### Programmatic Check

```go
if cfg.Bot.DemoMode {
    // Demo mode is enabled
}

if services.Demo.IsDemoUser(userID) {
    // This is the demo user
}
```

## Best Practices

### Development

✅ **DO**:

- Use demo mode for consistent test data
- Reset demo data before presentations
- Document any custom demo scenarios

❌ **DON'T**:

- Enable demo mode in production
- Modify demo user ID (999999999)
- Rely on demo data for real user testing

### Production

⚠️ **IMPORTANT**: Always disable demo mode in production:

```bash
DEMO_MODE=false
```

Demo mode is for development and demonstration only.

### Testing

Use demo mode in automated tests:

```bash
# In test environment
DEMO_MODE=true go test ./...
```

## Troubleshooting

### Demo data not appearing

**Check configuration:**

```bash
# Verify DEMO_MODE is set
grep DEMO_MODE .env

# Check logs for demo mode initialization
./shopogoda | grep -i demo
```

**Reset demo data manually:**

```plaintext
/demoreset  # As admin user
```

### Demo user already exists

Demo mode uses `FirstOrCreate` - it won't duplicate the demo user. If demo data seems incomplete:

1. Clear existing demo data: `/democlear`
2. Re-seed fresh data: `/demoreset`

### Permission denied for admin commands

Admin commands (`/demoreset`, `/democlear`) require admin role. Check user role:

```sql
SELECT telegram_id, username, role FROM users WHERE telegram_id = YOUR_ID;
```

Update role if needed:

```sql
UPDATE users SET role = 'admin' WHERE telegram_id = YOUR_ID;
```

## Demo Data Lifecycle

```plaintext
Bot Start (DEMO_MODE=true)
    ↓
Check if demo user exists
    ↓
Create/update demo user
    ↓
Generate 24h weather data
    ↓
Create alert configurations
    ↓
Create subscriptions
    ↓
Demo ready ✅
```

## Related Documentation

- [DEMO_SETUP.md](DEMO_SETUP.md) - Quick start guide
- [DEPLOYMENT.md](DEPLOYMENT.md) - Production deployment
- [CONFIGURATION.md](CONFIGURATION.md) - Complete configuration reference

## Support

For demo mode issues or questions:

- **GitHub Issues**: <https://github.com/valpere/shopogoda/issues>
