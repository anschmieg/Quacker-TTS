# Quacker - Multi-Provider TTS Client

A simple cross-platform desktop application for generating speech using multiple Text-to-Speech (TTS) APIs, including OpenAI TTS and Google Cloud TTS.

**Please note:**

1. This app has only been tested on macOS arm64.
   It should work on other platforms (Linux, Windows) as well, but there may be some issues. If you encounter any problems, please open an issue on the GitHub repository.
2. This is a personal project and not an official OpenAI product. It is not affiliated with or endorsed by OpenAI.

## Features

- **Multi-Provider Support**: Choose between OpenAI TTS and Google Cloud TTS APIs.
- **Custom Voice Configuration**: Use provider-specific voices and settings.
- **Adjustable Speech Speed**: Fine-tune playback speed for both providers.
- **Custom Instructions**: Provide custom instructions for voice generation (OpenAI).
- **Automatic Audio Saving**: Saves generated audio as MP3 files directly to your Downloads folder.
- **Smart Filename Generation**: Automatically generates filenames based on the first few words of input text (e.g., `Text_Hello_World.mp3`).
- **Secure Credential Management**: Uses environment variables or system keychain for API keys and configuration.
- **Intelligent Text Chunking**: Automatically splits large texts for optimal processing.

## Setup

Choose one or both TTS providers to configure:

### OpenAI TTS Setup

1.  **OpenAI API Key:** Get an API key from [OpenAI](https://platform.openai.com/api-keys).
2.  **Configure API Key:** Provide the key using one of these methods:
    - **Environment Variable:** Set `OPENAI_API_KEY` before launching the app.
    - **(Recommended) Settings Dialog:** Use the in-app Settings to securely store the key.
    - **Keychain/Keyring:** The app stores keys in your system's keychain using:
      - Service: `Quacker_OpenAI`
      - Account: `api_token`
    - **.env File:** Create `.env` in the app directory or home directory:
      ```
      OPENAI_API_KEY=your_actual_api_key_here
      ```

### Google Cloud TTS Setup

1.  **Google Cloud Project:** Create a project with [Cloud Text-to-Speech API](https://cloud.google.com/text-to-speech) enabled.
2.  **Authentication:** Install and configure [Google Cloud CLI](https://cloud.google.com/sdk/docs/install):
    ```bash
    # Install gcloud CLI, then authenticate
    gcloud auth login
    gcloud config set project YOUR_PROJECT_ID
    ```
3.  **Configure Project ID:** Provide your project ID using one of these methods:
    - **Environment Variable:** Set `GOOGLE_CLOUD_PROJECT` or `GCP_PROJECT`.
    - **Settings Dialog:** Use the in-app Settings to configure the project ID.
    - **gcloud config:** The app automatically detects your active gcloud project.
    - **.env File:** Add to your `.env` file:
      ```
      GOOGLE_CLOUD_PROJECT=your-project-id
      ```

### Provider Selection

- Use the **Settings** menu to configure providers and set your default.
- The **Provider** dropdown lets you switch between configured providers.
- Each provider has different available voices and features.

## Installation & Running

Download the latest release for your operating system and architecture from the [GitHub Releases page](https://github.com/anschmieg/easy-tts/releases).

### macOS (Universal: Intel & Apple Silicon)

1.  Download `Quacker-macOS-universal.zip`.
2.  Unzip the file. This will create `Quacker.app`.
3.  (Recommended) Move `Quacker.app` to your `/Applications` folder.
4.  **Important - Bypassing Gatekeeper (First Launch Only):**
    - macOS Gatekeeper will likely prevent the app from opening initially because it wasn't downloaded from the App Store and might not be signed with an Apple Developer ID.
    - To grant permission **only for this app** without changing your system security settings:
      - **Right-click** (or hold **Control** and click) on `Quacker.app`.
      - Select **Open** from the menu that appears.
      - You will see a warning dialog saying the developer cannot be verified. Click the **Open** button in this dialog.
    - You only need to do this the very first time you run the app. macOS will remember your choice for subsequent launches.

### Linux (x86_64 / arm64)

1.  Download the appropriate `.tar.gz` file for your architecture (`Quacker-linux-x86_64.tar.gz` or `Quacker-linux-arm64.tar.gz`).
2.  Extract the archive:
    ```bash
    tar xzvf Quacker-linux-*.tar.gz
    ```
3.  Make the binary executable (if needed):
    ```bash
    chmod +x Quacker
    ```
4.  Run the application:
    ```bash
    ./Quacker
    ```
5.  _Note:_ Ensure you have the necessary Fyne dependencies installed on your system (like `gcc`, `libgl1-mesa-dev`, `xorg-dev` on Debian/Ubuntu). Refer to the [Fyne documentation](https://developer.fyne.io/started/#prerequisites) for details.

### Windows (x86_64)

1.  Download `Quacker-windows-x86_64.zip`.
2.  Unzip the file.
3.  Run `Quacker.exe`.
4.  **Note - Windows SmartScreen:** Windows Defender SmartScreen might show a warning because the application is not commonly downloaded or signed. Click "More info" and then "Run anyway" to proceed.

## Supported Providers & Voices

### OpenAI TTS

- **Model:** `gpt-4o-mini-tts`
- **Default Voice:** `shimmer`
- **Supported Formats:** MP3, Opus, AAC, FLAC
- **Voice Options:** alloy, echo, fable, onyx, nova, shimmer

### Google Cloud TTS

- **Default Voice:** `de-DE-Chirp3-HD-Kore`
- **Supported Formats:** MP3, LINEAR16, OGG-Opus, MULAW, ALAW
- **Voice Options:** Hundreds of voices across 100+ languages
- **Advanced Features:** Neural voices, WaveNet voices, voice cloning

## Building from Source

If you prefer to build the application yourself:

1.  **Install Go:** Ensure you have Go installed (version 1.21 or later recommended).
2.  **Install Fyne Prerequisites:** Follow the Fyne documentation to install the necessary C compilers and graphics libraries for your OS: [Fyne Prerequisites](https://developer.fyne.io/started/#prerequisites).
3.  **Install Fyne CLI:**
    ```bash
    go install fyne.io/fyne/v2/cmd/fyne@latest
    ```
4.  **Clone the Repository:**
    ```bash
    git clone https://github.com/anschmieg/easy-tts.git
    cd easy-tts
    ```
5.  **Build:**
    ```bash
    go build .
    # Or package it (example for macOS universal):
    # fyne package -os darwin -arch universal -icon Icon.png
    ```

## Configuration Examples

### Using Multiple Providers

You can configure both providers and switch between them:

```bash
# Environment variables for both providers
export OPENAI_API_KEY="your-openai-key"
export GOOGLE_CLOUD_PROJECT="your-gcp-project"
export DEFAULT_TTS_PROVIDER="openai"  # or "google"
```

### .env File Example

```
OPENAI_API_KEY=sk-proj-abc123...
GOOGLE_CLOUD_PROJECT=my-tts-project
DEFAULT_TTS_PROVIDER=google
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Attribution:** _`Icon.png` includes an image created by [Vectors Market](https://thenounproject.com/creator/vectorsmarket/) from [Noun Project](https://thenounproject.com/) published under the [CC BY-3.0](https://creativecommons.org/licenses/by/3.0/) license._
