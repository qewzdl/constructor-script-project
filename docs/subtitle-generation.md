# Automatic subtitle generation

Constructor Script CMS can automatically generate WebVTT subtitles for uploaded course videos. The feature runs through the upload service and stores the generated subtitle file alongside the video attachment.

## Requirements

- An OpenAI account with access to the Whisper transcription model.
- A valid `OPENAI_API_KEY` scoped for Whisper usage.
- Network access from the CMS backend to `https://api.openai.com` (or your custom endpoint).

## Configuration

Subtitle generation is disabled by default. You can manage the feature from the admin panel under **Settings → Site → Subtitles**. Values saved through the interface are stored in the database, override any environment defaults, and take effect immediately without a restart.

The backend still honours environment variables as sensible defaults when the database does not contain overrides. The service turns on automatically when:

1. An `OPENAI_API_KEY` is present in the environment, **and**
2. No explicit `SUBTITLE_GENERATION_ENABLED` flag is set.

To forcefully enable or disable the feature via environment variables set `SUBTITLE_GENERATION_ENABLED` to `true` or `false` respectively.

### Minimum configuration

Add the following entries to your `.env` or deployment environment:

```env
OPENAI_API_KEY=sk-your-openai-api-key
# Optional overrides
# SUBTITLE_GENERATION_ENABLED=true
# SUBTITLE_PROVIDER=openai
# OPENAI_MODEL=whisper-1
# SUBTITLE_PREFERRED_NAME=lesson-subtitles
# SUBTITLE_LANGUAGE=en
# SUBTITLE_PROMPT=
# SUBTITLE_TEMPERATURE=0.0
```

With these values in place the application initialises the OpenAI subtitle generator during startup. Every new compatible video upload triggers an automatic Whisper transcription; the resulting `.vtt` file is saved using the configured preferred name when provided. When you later supply values through the admin interface they replace these defaults.

## Testing the setup

1. Restart the backend after updating environment variables so the configuration reloads.
2. Upload a supported video through the admin interface or API.
3. Inspect the upload result – a subtitle attachment with the title “Auto-generated subtitles” should be present. The file is stored inside the configured `UPLOAD_DIR`.

If subtitle generation fails, check the application logs. Errors from OpenAI (such as authentication failures or rate limits) are logged with the `Failed to initialise subtitle generator` or `transcription request failed` messages.

## Production deployments

The `deploy/quickstart.sh` helper writes commented subtitle settings into `deploy/.env.production`. Uncomment and adjust the lines to enable Whisper in production deployments created by the script.
