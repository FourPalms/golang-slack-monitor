# Slack Monitor

A lightweight Go application that monitors your Slack DMs and sends phone notifications via ntfy.sh.

## Features

- 🔔 Real-time monitoring of Slack direct messages
- 📱 Push notifications to your phone via ntfy.sh
- 🔄 Configurable polling interval (default: 60 seconds)
- 💾 Persistent state to avoid duplicate notifications
- 🔒 Simple manual token setup
- 🚀 No external dependencies (stdlib only)
- ⚡ Lightweight and fast (~400 lines of Go)

## Prerequisites

- Go 1.21 or higher
- A Slack workspace account
- ntfy.sh app on your phone (free, no account needed)

## Installation

### Build from source

```bash
git clone https://github.com/jeremyhunt/slack-monitor
cd slack-monitor
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

You need two tokens from your Slack browser session:

1. **Open Slack in your browser**: https://app.slack.com/client/YOUR_WORKSPACE
2. **Open DevTools**: Press `F12` (Windows/Linux) or `Cmd+Option+I` (Mac)
3. **Go to Application/Storage tab** → Cookies → https://app.slack.com
4. **Find the `d` cookie**:
   - Name: `d`
   - Value: This is your **xoxc token** (starts with `xoxc-`)
   - Copy the entire value

5. **Go to Network tab**:
   - Filter by "api" or "slack.com"
   - Look for any request to `slack.com/api/`
   - Click on a request and go to Headers
   - Find `Authorization: Bearer xoxd-...`
   - Copy the **xoxd token** (everything after "Bearer ")

### Step 3: Create Configuration File

Create `~/.slack-monitor/config.json`:

```bash
mkdir -p ~/.slack-monitor
cat > ~/.slack-monitor/config.json << 'EOF'
{
  "slack": {
    "xoxc_token": "xoxc-PASTE-YOUR-XOXC-TOKEN-HERE",
    "xoxd_token": "xoxd-PASTE-YOUR-XOXD-TOKEN-HERE",
    "workspace_id": "T02BJJRF2",
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

**Set secure permissions:**
```bash
chmod 600 ~/.slack-monitor/config.json
```

## Usage

### Run in foreground (for testing)

```bash
slack-monitor
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

### Run in background

```bash
nohup slack-monitor > ~/.slack-monitor/monitor.log 2>&1 &
```

Save the process ID:
```bash
echo $! > ~/.slack-monitor/monitor.pid
```

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
| `slack.xoxc_token` | string | Yes | - | Slack cookie token (starts with `xoxc-`) |
| `slack.xoxd_token` | string | Yes | - | Slack authorization token (starts with `xoxd-`) |
| `slack.workspace_id` | string | No | - | Your Slack workspace ID (optional) |
| `slack.poll_interval_seconds` | int | No | 60 | How often to check for new messages |
| `notifications.ntfy_topic` | string | Yes | - | Your ntfy.sh topic name |
| `monitor.dms_only` | bool | No | true | Monitor only DMs (future: channels/mentions) |

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

- **Config Management**: Loads and validates `config.json`
- **State Management**: Tracks last-checked timestamps in `state.json` (atomic writes)
- **Slack API Client**: Authenticates with xoxc/xoxd tokens, polls `conversations.list` and `conversations.history`
- **Notification Service**: Sends to ntfy.sh with rate limiting
- **Monitoring Loop**: Runs every N seconds (configurable), handles graceful shutdown on SIGTERM/SIGINT

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
