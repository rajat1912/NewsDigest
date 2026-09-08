# News Agent - Quick Start Guide 🚀

Your personalized daily news-agent is ready! Here's what's been set up:

## What You Have

✅ **Complete News Agent with**:
- News fetching from RSS feeds
- Personalized preference filtering
- Article deduplication
- Telegram bot integration
- GitHub Actions scheduled delivery
- JSON-based state tracking

## Project Structure

```
news-agent/
├── main.go                          # Orchestration & scheduling
├── internal/
│   ├── config/config.go            # Config loading
│   ├── news/fetcher.go             # RSS feed fetching
│   ├── telegram/client.go          # Telegram bot integration
│   ├── summarizer/summarizer.go    # Article curation (placeholder for OpenAI)
│   └── storage/store.go            # Deduplication tracking
├── .github/workflows/
│   └── daily-news.yml              # GitHub Actions scheduler
├── config.yaml                      # Your preferences
├── .env.example                     # Environment template
├── sent_articles.json               # Auto-managed dedup store
└── README.md                        # Full documentation
```

## Next Steps

### 1. Get Your Telegram Bot Token
Visit [@BotFather](https://t.me/BotFather) on Telegram:
- Send `/newbot`
- Follow prompts to create a bot
- Copy the **HTTP API token**

### 2. Get Your Telegram Chat ID
Send any message to your bot, then visit:
```
https://api.telegram.org/bot<YOUR_TOKEN>/getUpdates
```
Find your chat ID in the response.

### 3. Set Up GitHub Secrets
Push to GitHub, then go to **Settings → Secrets and variables → Actions**:

Add these secrets:
- `TELEGRAM_BOT_TOKEN` — Your bot token
- `TELEGRAM_CHAT_ID` — Your chat ID ̑
- `OPENAI_API_KEY` — Your OpenAI key (optional, for now)
- `NEWSAPI_API_KEY` — Your NewsAPI key (optional)

### 4. Customize Your Preferences
Edit `config.yaml`:
- Add your topics (tech, sports, etc.)
- Add RSS feeds you like
- Set number of articles per delivery

### 5. Deploy to GitHub
```bash
git add .
git commit -m "Initialize news-agent"
git push
```

GitHub Actions will now run daily at **8 AM IST** (adjust in `.github/workflows/daily-news.yml` for your timezone).

## Testing Locally

Run once to test:
```bash
export TELEGRAM_BOT_TOKEN="your-token"
export TELEGRAM_CHAT_ID="your-chat-id"
./news-agent --config config.yaml --once
```

Run with scheduler locally:
```bash
./news-agent --config config.yaml  # Ctrl+C to stop
```

## Cron Schedule Reference

GitHub Actions runs in UTC. Adjust `.github/workflows/daily-news.yml`:

| Time | UTC Cron | Cron |
|------|----------|------|
| 8 AM IST | 2:30 AM UTC | `30 2 * * *` |
| 8 AM EST | 1 PM UTC | `0 13 * * *` |
| 8 AM PST | 4 PM UTC | `0 16 * * *` |
| 8 AM UTC | 8 AM UTC | `0 8 * * *` |

Use [crontab.guru](https://crontab.guru) to generate your expression.

## How It Works

1. **GitHub Actions** triggers daily at your scheduled time
2. **Fetch** articles from configured RSS feeds
3. **Filter** by your topics & preferences
4. **Check** sent_articles.json for duplicates
5. **Summarize** & curate top articles
6. **Send** formatted message to Telegram
7. **Track** sent articles in sent_articles.json

## Key Features

- **Smart Dedup**: Never sends the same article twice (30-day history)
- **Preference Filtering**: Only relevant articles make the cut
- **Serverless**: Runs on GitHub Actions (no server costs)
- **Git-Backed State**: Dedup state stored in git repo
- **Configurable**: Easy YAML customization
- **Reliable**: Error logging & automatic retries

## Troubleshooting

**No messages arriving?**
- Check GitHub Actions logs
- Verify `TELEGRAM_BOT_TOKEN` and `TELEGRAM_CHAT_ID` are set
- Make sure the bot has permission to message you

**Wrong delivery time?**
- Adjust cron expression in `.github/workflows/daily-news.yml`
- Remember: GitHub runs in UTC!

**Duplicate articles?**
- `sent_articles.json` might not be committing
- Manually run a test with `--once` flag locally

**No articles found?**
- Check your RSS feeds are active
- Verify topic keywords match feed content
- Try adding more diverse feeds

## Files Explained

- **main.go** — Entry point, handles CLI flags & job orchestration
- **config.go** — Loads YAML + env vars
- **fetcher.go** — RSS feed parser & preference filter
- **client.go** — Telegram bot API wrapper
- **summarizer.go** — Article curation logic
- **store.go** — JSON deduplication storage
- **daily-news.yml** — GitHub Actions workflow

## Future Enhancements

Consider adding:
- OpenAI integration for AI curation (update summarizer.go)
- Multiple Telegram channels/users
- Web dashboard for preferences
- Email delivery option
- Advanced filtering (sentiment, date, source)

## Support

Check the [full README.md](README.md) for complete documentation!

---

**Ready?** Push to GitHub and watch your daily news arrive at 8 AM! 🎉
