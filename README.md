# News Agent 📰

A daily news aggregator that fetches, curates, and delivers personalized news summaries to your Telegram bot every morning at 8 AM using GitHub Actions.

## Features

✅ **Personalized News Curation** — Filter articles by your preferred topics  
✅ **AI-Powered Summarization** — Use OpenAI to summarize and rank articles  
✅ **Multiple News Sources** — Fetch from RSS feeds and NewsAPI  
✅ **Duplicate Prevention** — Tracks sent articles to avoid repetition  
✅ **Serverless Scheduling** — GitHub Actions runs daily at 8 AM (configurable timezone)  
✅ **Easy Configuration** — YAML-based preferences and environment variables  

## Architecture

```
news-agent/
├── cmd/                    # Entry point
├── internal/
│   ├── config/            # Configuration loading (YAML + env vars)
│   ├── news/              # News fetching (RSS + NewsAPI)
│   ├── telegram/          # Telegram bot integration
│   ├── summarizer/        # OpenAI-powered curation
│   └── storage/           # Duplicate tracking (JSON-based)
├── .github/workflows/
│   └── daily-news.yml     # GitHub Actions scheduler
├── config.yaml            # User preferences
└── sent_articles.json     # Persistence layer
```

### Flow

1. **Fetch** — Pull articles from configured RSS feeds and NewsAPI
2. **Filter** — Match against user preferences (topics, languages, countries)
3. **Summarize** — Send to OpenAI for curation and summarization
4. **Deduplicate** — Check against `sent_articles.json` to prevent repeats
5. **Send** — Format and deliver to Telegram
6. **Track** — Save sent articles for future deduplication

## Setup

### 1. Clone & Prerequisites

```bash
git clone <your-repo>
cd news-agent
```

Requires:
- Go 1.26.1+
- Telegram bot token (get from [@BotFather](https://t.me/BotFather))
- OpenAI API key (from [platform.openai.com](https://platform.openai.com))
- NewsAPI key (optional, from [newsapi.org](https://newsapi.org))

### 2. Get Your Telegram Chat ID

Send a message to your bot, then visit:
```
https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates
```

Find your chat ID in the JSON response (`message.chat.id`).

### 3. Configure Locally

Copy `.env.example` to `.env` and fill in your API keys:
```bash
cp .env.example .env
```

Edit `config.yaml` to customize:
- **topics** — News categories you care about (e.g., `["technology", "golang"]`)
- **rss** — RSS feed URLs
- **num_articles** — How many articles per daily delivery
- **languages** — Preferred languages for articles

### 4. Test Locally

```bash
# Run once to test
go run main.go --config config.yaml --once

# Or build and run
go build -o news-agent
./news-agent --config config.yaml --once
```

### 5. Deploy to GitHub Actions

#### Create Repository Secrets

Go to **Settings → Secrets and variables → Actions** and add:
- `OPENAI_API_KEY` — Your OpenAI API key
- `TELEGRAM_BOT_TOKEN` — Your Telegram bot token
- `TELEGRAM_CHAT_ID` — Your Telegram chat ID
- `NEWSAPI_API_KEY` — (Optional) NewsAPI key

#### Adjust Scheduling (Important!)

The workflow defaults to **8 AM IST** (2:30 AM UTC). Edit `.github/workflows/daily-news.yml` to match your timezone:

**Common timezones:**
- 8 AM IST (UTC+5:30) → `30 2 * * *` ✅ (default)
- 8 AM EST (UTC-5) → `0 13 * * *`
- 8 AM PST (UTC-8) → `0 16 * * *`
- 8 AM UTC → `0 8 * * *`

Find more: [crontab.guru](https://crontab.guru/)

#### Push & Enable

```bash
git add .
git commit -m "Initial news-agent setup"
git push
```

GitHub Actions will automatically start running daily at your scheduled time!

## Local Testing

To test the workflow locally (requires [act](https://github.com/nektos/act)):

```bash
# Create .env file with secrets
echo "OPENAI_API_KEY=<your-key>" > .env
echo "TELEGRAM_BOT_TOKEN=<your-token>" >> .env
echo "TELEGRAM_CHAT_ID=<your-chat-id>" >> .env

# Run workflow locally
act --env-file .env
```

## Customization

### Add More RSS Feeds

Edit `config.yaml`:
```yaml
rss:
  feeds:
    - "https://feeds.arstechnica.com/arstechnica/index"
    - "https://news.ycombinator.com/rss"
    - "https://yourfeed.com/rss"
```

### Adjust Article Count

Change `num_articles` in `config.yaml` (default: 5).

### Change Topics

Personalize `preferences.topics` in `config.yaml` for better filtering.

### Modify OpenAI Model

Use `gpt-4o` for better quality (costs more), or `gpt-4o-mini` for budget-friendly option.

## Troubleshooting

**No articles sent?**
- Check that your RSS feeds are valid and have recent articles
- Verify API keys are correctly set in GitHub Secrets
- Check GitHub Actions logs for errors

**Duplicate articles being sent?**
- Ensure `sent_articles.json` is committed to your repository
- GitHub Actions auto-commits this file after each run

**Telegram messages not arriving?**
- Verify `TELEGRAM_CHAT_ID` is correct
- Make sure the bot is in the chat/channel

**Wrong delivery time?**
- Adjust cron expression in `.github/workflows/daily-news.yml`
- Remember: GitHub Actions runs in UTC!

## File Structure

- `main.go` — Orchestrates all components and schedules jobs
- `internal/config/config.go` — Loads YAML + environment variables
- `internal/news/fetcher.go` — Fetches from RSS and NewsAPI, filters by preferences
- `internal/telegram/client.go` — Sends formatted messages to Telegram
- `internal/summarizer/summarizer.go` — OpenAI integration for curation
- `internal/storage/store.go` — JSON-based deduplication storage
- `.github/workflows/daily-news.yml` — GitHub Actions workflow
- `config.yaml` — Your preferences (topics, feeds, article count)
- `sent_articles.json` — Auto-generated, tracks sent articles

## Advanced: Running Locally with Scheduler

To test cron scheduling locally:

```bash
# Runs with scheduler (Ctrl+C to stop)
go run main.go --config config.yaml
```

This starts a local cron scheduler that runs daily at 8 AM.

## License

MIT

## Support

For issues or improvements, open a GitHub issue or PR!
