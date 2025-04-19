# Quacker - OpenAI TTS Client

A simple cross-platform desktop application for generating speech using the OpenAI Text-to-Speech (TTS) API (with model *gpt-4o-mini-tts*). 

**Please note:** 
1. This app has only been tested on macOS arm64. 
   It should work on other platforms (Linux, Windows) as well, but there may be some issues. If you encounter any problems, please open an issue on the GitHub repository.
2. This is a personal project and not an official OpenAI product. It is not affiliated with or endorsed by OpenAI.

## Features

*   Generate speech from text using custom-defined OpenAI voices.
*   Adjust speech speed.
*   Provide custom instructions for the voice.
*   Saves generated audio as an MP3 file directly to your Downloads folder.
*   Filename is automatically generated based on the first few words of the input text (e.g., `Text_Hello_World.mp3`).
*   Securely uses your OpenAI API key via environment variables or system keychain.

## Setup

1.  **OpenAI API Key:** You need an API key from OpenAI.
2.  **Configure API Key:** Provide the key to the application in one of these ways:
    *   **Environment Variable:** Set the `OPENAI_API_KEY` environment variable before launching the app.
    *   **(Recommended) Keychain/Keyring:** Store the key securely in your system's keychain or keyring. The application looks for it using:
        *   Service/Application: `OpenAI_TTS`
        *   Account/User: `api_token`
        *(You might need a separate tool or script to add the key to your keychain initially)*.
    *   **.env File:** Create a file named `.env` in the same directory as the application *or* in your user home directory and add the line:
        ```
        OPENAI_API_KEY=your_actual_api_key_here
        ```

## Installation & Running

Download the latest release for your operating system and architecture from the [GitHub Releases page](https://github.com/anschmieg/easy-tts/releases).

### macOS (Universal: Intel & Apple Silicon)

1.  Download `Quacker-macOS-universal.zip`.
2.  Unzip the file. This will create `Quacker.app`.
3.  (Recommended) Move `Quacker.app` to your `/Applications` folder.
4.  **Important - Bypassing Gatekeeper (First Launch Only):**
    *   macOS Gatekeeper will likely prevent the app from opening initially because it wasn't downloaded from the App Store and might not be signed with an Apple Developer ID.
    *   To grant permission **only for this app** without changing your system security settings:
        *   **Right-click** (or hold **Control** and click) on `Quacker.app`.
        *   Select **Open** from the menu that appears.
        *   You will see a warning dialog saying the developer cannot be verified. Click the **Open** button in this dialog.
    *   You only need to do this the very first time you run the app. macOS will remember your choice for subsequent launches.

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
5.  *Note:* Ensure you have the necessary Fyne dependencies installed on your system (like `gcc`, `libgl1-mesa-dev`, `xorg-dev` on Debian/Ubuntu). Refer to the [Fyne documentation](https://developer.fyne.io/started/#prerequisites) for details.

### Windows (x86_64)

1.  Download `Quacker-windows-x86_64.zip`.
2.  Unzip the file.
3.  Run `Quacker.exe`.
4.  **Note - Windows SmartScreen:** Windows Defender SmartScreen might show a warning because the application is not commonly downloaded or signed. Click "More info" and then "Run anyway" to proceed.

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

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

**Attribution:** *`Icon.png` includes an image created by [Vectors Market](https://thenounproject.com/creator/vectorsmarket/) from [Noun Project](https://thenounproject.com/) published under the [CC BY-3.0](https://creativecommons.org/licenses/by/3.0/) license.*