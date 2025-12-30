# Slack Monitor

A lightweight Go application that monitors your Slack DMs and sends phone notifications via ntfy.sh.

## Features

- 🔔 Real-time monitoring of Slack direct messages
- 📱 Push notifications to your phone via ntfy.sh
- 🔄 Configurable polling interval (default: 60 seconds)
- 💾 Persistent state to avoid duplicate notifications
- 🔒 Simple manual token setup
- 🚀 No external dependencies (stdlib only)
- ⚡ Lightweight and fast (clean package architecture)

## Prerequisites

- Go 1.21 or higher
- A Slack workspace account
- ntfy.sh app on your phone (free, no account needed)

## Quick Start

For experienced developers:

1. **Clone and build**:
   ```bash
   git clone git@github.com:FourPalms/golang-slack-monitor.git
   cd golang-slack-monitor
   make build
   ```

2. **Extract Slack tokens** (see Step 2 below for detailed instructions)

3. **Create config** (see `config.example.json` for reference):
   ```bash
   mkdir -p ~/.slack-monitor
   cp config.example.json ~/.slack-monitor/config.json
   # Edit with your tokens and ntfy topic
   chmod 600 ~/.slack-monitor/config.json
   ```

4. **Run**:
   ```bash
   ./slack-monitor
   ```

## Installation

### Build from source

```bash
git clone git@github.com:FourPalms/golang-slack-monitor.git
cd golang-slack-monitor
make build
```

### Install to ~/bin

```bash
make install
```

Make sure `~/bin` is in your PATH:
```bash
export PATH="$HOME/bin:$PATH"
```

## Setup

### Step 1: Get ntfy.sh Topic

1. Install the ntfy app on your phone:
   - iOS: https://apps.apple.com/app/ntfy/id1625396347
   - Android: https://play.google.com/store/apps/details?id=io.heckel.ntfy
2. Choose a random topic name (this is your "password")
   - Example: `my-slack-monitor-89234792`
   - **Important**: Use a random suffix to prevent others from guessing your topic

### Step 2: Extract Slack Tokens

You need two tokens from your Slack browser session. This app uses "stealth mode" authentication (same method as slack-mcp-server).

**Important**: Both tokens work together - you need BOTH for authentication to work.

1. **Open Slack in your browser**: https://app.slack.com/client/YOUR_WORKSPACE

2. **Open DevTools**: Press `F12` (Windows/Linux) or `Cmd+Option+I` (Mac)

3. **Extract xoxd token** (Cookie "d"):
   - Go to **Application** (Chrome) or **Storage** (Firefox) tab
   - Navigate to Cookies → https://app.slack.com
   - Find the cookie named **`d`**
   - Copy its **Value** - this is your **xoxd token** (starts with `xoxd-`)
   - ⚠️ **Note**: The cookie is named "d" but contains the "xoxd" token (not "xoxc")

4. **Extract xoxc token** (API requests):
   - Go to **Network** tab in DevTools
   - Refresh the Slack page or click around to generate some API traffic
   - Filter by "api" or search for `slack.com/api/`
   - Click on any API request (e.g., `conversations.list`, `users.info`)
   - Look at the **Request URL** or **Query String Parameters**
   - Find the `token` parameter - this is your **xoxc token** (starts with `xoxc-`)
   - Copy the entire token value

**Why both tokens?**
- **xoxd token**: Goes in the "d" cookie for session authentication
- **xoxc token**: Goes in API request parameters for user authentication
- The app uses both together to authenticate as your user without creating a Slack app

### Step 3: Create Configuration File

Create `~/.slack-monitor/config.json` (see `config.example.json` in the repo for reference):

```bash
mkdir -p ~/.slack-monitor
cat > ~/.slack-monitor/config.json << 'EOF'
{
  "slack": {
    "xoxc_token": "xoxc-PASTE-YOUR-XOXC-TOKEN-HERE",
    "xoxd_token": "xoxd-PASTE-YOUR-XOXD-TOKEN-HERE",
    "poll_interval_seconds": 60
  },
  "notifications": {
    "ntfy_topic": "your-topic-name-here"
  },
  "monitor": {
    "dms_only": true
  }
}
EOF
```

**Replace the placeholder values:**
- `xoxc-PASTE-YOUR-XOXC-TOKEN-HERE` → Your xoxc token from Step 2.4
- `xoxd-PASTE-YOUR-XOXD-TOKEN-HERE` → Your xoxd token from Step 2.3
- `your-topic-name-here` → Your ntfy.sh topic from Step 1

**Set secure permissions:**
```bash
chmod 600 ~/.slack-monitor/config.json
```

## Usage

### Run in foreground (recommended for testing)

**Using make (keeps Mac awake automatically):**
```bash
make run
```

**Or run directly:**
```bash
./slack-monitor
```

You'll see output like:
```
2025/12/30 11:00:00 Slack Monitor starting...
2025/12/30 11:00:00 Config loaded successfully (poll interval: 60s)
2025/12/30 11:00:00 State loaded successfully (0 conversations tracked)
2025/12/30 11:00:00 Starting monitoring loop...
2025/12/30 11:00:00 Checking for new messages...
2025/12/30 11:00:00 Checking 5 DM conversation(s)
2025/12/30 11:00:01 Check cycle complete
```

⚠️ **Important**: The monitor cannot run when your Mac is in sleep mode. Use `make run` (which uses `caffeinate`) to keep your Mac awake while monitoring, or see the [Run as a service](#run-as-a-service-macos---launchd) section below.

### Run in background (keeps Mac awake)

```bash
nohup caffeinate -i ./slack-monitor > ~/.slack-monitor/monitor.log 2>&1 &
```

Save the process ID:
```bash
echo $! > ~/.slack-monitor/monitor.pid
```

**Note**: Using `caffeinate -i` prevents your Mac from sleeping while the monitor runs. The display can still sleep to save power.

### Stop the monitor

```bash
kill $(cat ~/.slack-monitor/monitor.pid)
```

Or:
```bash
pkill slack-monitor
```

### Run as a service (macOS - launchd)

Create `~/Library/LaunchAgents/com.user.slack-monitor.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.user.slack-monitor</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/bin/caffeinate</string>
        <string>-i</string>
        <string>/Users/YOUR_USERNAME/bin/slack-monitor</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/YOUR_USERNAME/.slack-monitor/monitor.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/YOUR_USERNAME/.slack-monitor/monitor.log</string>
</dict>
</plist>
```

**Replace `YOUR_USERNAME` with your actual username**.

**Note**: This configuration uses `caffeinate -i` to keep your Mac awake while the monitor runs. The display can still sleep to save power.

Load the service:
```bash
launchctl load ~/Library/LaunchAgents/com.user.slack-monitor.plist
```

Check status:
```bash
launchctl list | grep slack-monitor
```

Unload the service:
```bash
launchctl unload ~/Library/LaunchAgents/com.user.slack-monitor.plist
```

## Configuration

### Config file: `~/.slack-monitor/config.json`

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `slack.xoxc_token` | string | **Yes** | - | Slack user token (starts with `xoxc-`). Found in API request `token` parameter. |
| `slack.xoxd_token` | string | **Yes** | - | Slack session token (starts with `xoxd-`). Found in cookie "d". |
| `slack.poll_interval_seconds` | int | No | 60 | How often to check for new messages (in seconds). Minimum: 30, recommended: 60-300. |
| `notifications.ntfy_topic` | string | **Yes** | - | Your ntfy.sh topic name. Use a random suffix for security. |
| `monitor.dms_only` | bool | No | true | Monitor only DMs. Currently only `true` is supported. |

### State file: `~/.slack-monitor/state.json`

Automatically created and managed. Tracks the last checked timestamp for each conversation to avoid duplicate notifications.

**Do not edit manually** unless you know what you're doing.

## Troubleshooting

### "invalid_auth" error

Your tokens have expired. Slack tokens typically last several months but may expire sooner. Re-extract tokens following Step 2.

### No notifications received

1. **Test ntfy.sh directly**:
   ```bash
   curl -d "Test message" ntfy.sh/your-topic-name
   ```
   You should receive a notification immediately.

2. **Check logs** for errors:
   ```bash
   tail -f ~/.slack-monitor/monitor.log
   ```

3. **Verify config** file has correct topic:
   ```bash
   cat ~/.slack-monitor/config.json | grep ntfy_topic
   ```

### Too many notifications

The monitor has built-in rate limiting (2 seconds between notifications), but if you're getting too many:

1. **Increase poll interval** in config:
   ```json
   "poll_interval_seconds": 300
   ```
   (This checks every 5 minutes instead of every 60 seconds)

2. **Check state file** to see which conversations are tracked:
   ```bash
   cat ~/.slack-monitor/state.json
   ```

### Notifications for old messages

On first run, the monitor starts tracking from "now" to avoid spamming you with old messages. If you're still getting old messages:

1. Stop the monitor
2. Delete state file: `rm ~/.slack-monitor/state.json`
3. Start the monitor again

## Development

### Build

```bash
make build
```

### Run tests

```bash
make test
```

### Clean

```bash
make clean
```

## Architecture

Follows Ben Johnson's Standard Package Layout for clean separation of concerns:

```
slack-monitor/
├── monitor.go              # Domain types & interfaces
├── slack/
│   ├── client.go           # Slack API client implementation
│   └── types.go            # API response types
├── notification/
│   └── service.go          # ntfy.sh notification service
├── storage/
│   └── state.go            # File-based state persistence
└── cmd/slack-monitor/
    └── main.go             # Main entry point (dependency wiring)
```

**Key Components:**
- **Domain Layer** (`monitor.go`): Core types (Message, Conversation, State) and interfaces (SlackClient, Notifier, StateStore)
- **Slack Client** (`slack/`): Authenticates with xoxc/xoxd tokens, polls `conversations.list` and `conversations.history`
- **Notification Service** (`notification/`): Sends to ntfy.sh with rate limiting (2-second minimum)
- **State Storage** (`storage/`): Atomic writes to `state.json`, tracks last-checked timestamps
- **Main Package** (`cmd/slack-monitor/`): Loads config, wires dependencies, handles graceful shutdown (SIGTERM/SIGINT)

## Security

- **Tokens**: Stored in plain text in `config.json`. Set file permissions to `600` (owner read/write only).
- **Token lifespan**: Slack tokens typically last months. Re-extract when they expire.
- **ntfy.sh**: No authentication. Use a random topic name that others cannot guess.
- **Rate limiting**: 2-second minimum between notifications to avoid spam.

## Known Limitations

- **DMs only**: Currently only monitors direct messages (no channels or @mentions yet)
- **No threading**: Monitors top-level messages only
- **Token expiration**: No automatic token refresh (manual re-extraction required)
- **Single workspace**: Monitors one Slack workspace at a time

## Future Enhancements

- [ ] Monitor @mentions in channels
- [ ] Keyword filtering
- [ ] Multiple workspaces support
- [ ] Web UI dashboard
- [ ] Docker container
- [ ] Automatic token refresh (OAuth flow)

## License

MIT License - feel free to use and modify.

## Contributing

Pull requests welcome! Please include tests for new features.

## Author

Built with ❤️ by Jeremy Hunt
