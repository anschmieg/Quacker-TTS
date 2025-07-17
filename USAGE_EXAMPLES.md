# Usage Examples

This document provides practical examples of how to use Quacker with different TTS providers.

## Quick Start

### 1. OpenAI TTS (Simplest Setup)

```bash
# Set your OpenAI API key
export OPENAI_API_KEY="sk-proj-your-key-here"

# Run the app
./easy-tts
```

1. The app will automatically detect your OpenAI configuration
2. Select "openai" from the Provider dropdown (should be selected by default)
3. Enter your text in the Input Text area
4. Click Submit
5. Audio file will be saved to your Downloads folder

### 2. Google Cloud TTS

```bash
# Install and configure gcloud CLI
gcloud auth login
gcloud config set project your-project-id

# Enable the Text-to-Speech API
gcloud services enable texttospeech.googleapis.com

# Run the app
./easy-tts
```

1. Select "google" from the Provider dropdown
2. The app will automatically detect your gcloud project
3. Enter your text and click Submit

## Configuration Examples

### Environment Variables

Create a `.env` file in your home directory or app directory:

```env
# OpenAI Configuration
OPENAI_API_KEY=sk-proj-abc123def456...

# Google Cloud Configuration
GOOGLE_CLOUD_PROJECT=my-tts-project

# Set default provider
DEFAULT_TTS_PROVIDER=openai
```

### Using Both Providers

```bash
# Configure both providers
export OPENAI_API_KEY="your-openai-key"
export GOOGLE_CLOUD_PROJECT="your-gcp-project"
export DEFAULT_TTS_PROVIDER="google"
```

You can then switch between providers using the dropdown in the app.

## Voice Examples

### OpenAI Voices

Popular OpenAI voices and their characteristics:

- **alloy** - Neutral, clear voice
- **echo** - Male-sounding voice
- **fable** - Female-sounding voice
- **onyx** - Deep male voice
- **nova** - Female voice with energy
- **shimmer** - Soft female voice (default)

Example usage:
1. Set Provider to "openai"
2. Set Voice to "nova"
3. Adjust speed (0.5-2.0, default 1.125)

### Google Cloud Voices

Google offers hundreds of voices. Some examples:

**English:**
- `en-US-Neural2-A` - Female, natural
- `en-US-Neural2-C` - Male, natural
- `en-US-Studio-M` - Male, expressive

**German:**
- `de-DE-Neural2-A` - Female
- `de-DE-Neural2-B` - Male
- `de-DE-Chirp3-HD-Kore` - High-quality female (default)

**Spanish:**
- `es-ES-Neural2-A` - Female
- `es-MX-Neural2-B` - Mexican Spanish, Male

Example usage:
1. Set Provider to "google"
2. Set Voice to "en-US-Neural2-A"
3. Adjust speed (0.25-4.0, default 1.0)

## Speed Settings

### OpenAI TTS
- Range: 0.25 to 4.0
- Default: 1.125 (slightly faster than normal)
- Recommended: 0.75-1.5 for most content

### Google Cloud TTS
- Range: 0.25 to 4.0
- Default: 1.0 (normal speed)
- Recommended: 0.8-1.3 for most content

## Text Input Tips

### Formatting for Better Results

**Good:**
```
# Chapter 1: Introduction

This is the beginning of our story. We'll explore several key concepts:

1. First concept
2. Second concept
3. Third concept

Let's dive deeper into each one.
```

**Avoid:**
```
CH1:intro!!!this is beginning we'll explore concepts:1st,2nd,3rd...
```

### Large Text Handling

Both providers automatically split large texts:

- **OpenAI**: Splits at ~2000 tokens (~1500 words)
- **Google**: Splits at ~5000 characters (~800 words)

The app intelligently splits at sentence boundaries when possible.

### Special Instructions (OpenAI Only)

Use the Instructions field for voice customization:

```
Speak in a calm, educational tone. Emphasize technical terms slightly.
Pause briefly after each bullet point. Use a friendly but professional voice.
```

## File Output

### Automatic Naming

Files are automatically named based on your input text:

Input: "Hello world, this is a test"
Output: `Text_Hello_world.mp3`

### Supported Formats

**OpenAI:**
- MP3 (default)
- Opus
- AAC
- FLAC

**Google Cloud:**
- MP3 (default)
- LINEAR16 (WAV)
- OGG-Opus
- MULAW
- ALAW

### File Location

All files are saved to your Downloads folder:
- macOS: `/Users/username/Downloads/`
- Linux: `/home/username/Downloads/`
- Windows: `C:\Users\username\Downloads\`

## Troubleshooting Common Issues

### "No providers configured"

**Problem:** App shows no providers available.

**Solutions:**
1. Check your API keys: `echo $OPENAI_API_KEY`
2. Check gcloud config: `gcloud config list`
3. Use Settings menu to configure manually
4. Check `.env` file format

### "Authentication failed" (Google)

**Problem:** Can't authenticate with Google Cloud.

**Solutions:**
1. Run `gcloud auth login`
2. Set correct project: `gcloud config set project PROJECT-ID`
3. Enable API: `gcloud services enable texttospeech.googleapis.com`
4. Check billing is enabled on your project

### "API key not found" (OpenAI)

**Problem:** OpenAI authentication fails.

**Solutions:**
1. Verify key format: Should start with `sk-proj-` or `sk-`
2. Check key permissions on OpenAI dashboard
3. Try setting key via Settings menu
4. Check for extra spaces or newlines

### "Text too long"

**Problem:** Text exceeds provider limits.

**Solutions:**
- The app automatically splits text, so this shouldn't happen
- If it does, try breaking your text into smaller paragraphs
- Check for very long single sentences

### Audio quality issues

**Problem:** Generated audio sounds robotic or unclear.

**Solutions:**
1. Try different voices
2. Adjust speed settings
3. Add punctuation for better pacing
4. Use the Instructions field (OpenAI) for voice guidance

### Performance issues

**Problem:** App is slow or unresponsive.

**Solutions:**
1. Check internet connection
2. Try smaller text chunks
3. Close other applications
4. Check provider API status pages

## Advanced Usage

### Batch Processing

For multiple files, you can:
1. Process one text at a time through the GUI
2. Use different voices/speeds for variety
3. Switch providers for different content types

### Voice Mixing

Create variety by:
1. Using OpenAI for conversational content
2. Using Google for technical/formal content
3. Trying different speeds for emphasis

### Workflow Integration

1. **Content Creation**: Use fast, clear voices for drafts
2. **Final Production**: Use higher-quality voices with careful speed tuning
3. **Accessibility**: Use slower speeds and clear pronunciation

## API Usage Limits

### OpenAI TTS
- Rate limit: ~50 requests/minute
- Character limit: No hard limit (app handles chunking)
- Cost: ~$15 per 1M characters

### Google Cloud TTS
- Rate limit: 300 requests/minute (Standard), 1000/minute (Premium)
- Character limit: 5000 per request (app handles chunking)
- Cost: Free tier: 1M characters/month, then ~$4 per 1M characters

## Getting Help

1. Check this file for common solutions
2. Verify provider API status pages
3. Test with simple, short text first
4. Check the main README.md for setup instructions
5. Open an issue on GitHub with:
   - Your operating system
   - Provider being used
   - Error messages (remove sensitive data)
   - Steps to reproduce the issue
