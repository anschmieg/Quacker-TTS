#!/bin/bash

# Test script for Quacker TTS providers
# This script helps you verify that your TTS providers are configured correctly

set -e

echo "🎤 Quacker TTS Provider Test Script"
echo "=================================="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test text
TEST_TEXT="Hello, this is a test of the text-to-speech system. The quick brown fox jumps over the lazy dog."

# Function to print colored output
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

print_success() {
    print_status $GREEN "✅ $1"
}

print_error() {
    print_status $RED "❌ $1"
}

print_warning() {
    print_status $YELLOW "⚠️  $1"
}

print_info() {
    print_status $BLUE "ℹ️  $1"
}

# Test OpenAI configuration
test_openai() {
    echo
    print_info "Testing OpenAI TTS configuration..."

    if [ -z "$OPENAI_API_KEY" ]; then
        print_error "OPENAI_API_KEY environment variable not set"
        print_info "Set it with: export OPENAI_API_KEY='your-key-here'"
        return 1
    fi

    # Check key format
    if [[ $OPENAI_API_KEY == sk-* ]]; then
        print_success "OpenAI API key format looks correct"
    else
        print_warning "OpenAI API key format might be incorrect (should start with 'sk-')"
    fi

    # Test API connection (simple curl test)
    print_info "Testing OpenAI API connection..."
    if command -v curl >/dev/null 2>&1; then
        local response
        response=$(curl -s -w "%{http_code}" -o /dev/null \
            -H "Authorization: Bearer $OPENAI_API_KEY" \
            -H "Content-Type: application/json" \
            -d '{
                "model": "gpt-4o-mini-tts",
                "voice": "shimmer",
                "speed": 1.0,
                "input": "test",
                "response_format": "mp3"
            }' \
            https://api.openai.com/v1/audio/speech)

        if [ "$response" = "200" ]; then
            print_success "OpenAI API connection successful"
            return 0
        else
            print_error "OpenAI API connection failed (HTTP $response)"
            return 1
        fi
    else
        print_warning "curl not available, skipping API connection test"
        return 0
    fi
}

# Test Google Cloud configuration
test_google() {
    echo
    print_info "Testing Google Cloud TTS configuration..."

    # Check if gcloud is installed
    if ! command -v gcloud >/dev/null 2>&1; then
        print_error "gcloud CLI not installed"
        print_info "Install from: https://cloud.google.com/sdk/docs/install"
        return 1
    fi

    print_success "gcloud CLI is installed"

    # Check if authenticated
    local current_account
    current_account=$(gcloud auth list --filter=status:ACTIVE --format="value(account)" 2>/dev/null || echo "")

    if [ -z "$current_account" ]; then
        print_error "Not authenticated with gcloud"
        print_info "Run: gcloud auth login"
        return 1
    fi

    print_success "Authenticated as: $current_account"

    # Check project configuration
    local project_id
    project_id=$(gcloud config get-value project 2>/dev/null || echo "")

    if [ -z "$project_id" ]; then
        # Try environment variables
        if [ -n "$GOOGLE_CLOUD_PROJECT" ]; then
            project_id="$GOOGLE_CLOUD_PROJECT"
            print_info "Using project from GOOGLE_CLOUD_PROJECT: $project_id"
        elif [ -n "$GCP_PROJECT" ]; then
            project_id="$GCP_PROJECT"
            print_info "Using project from GCP_PROJECT: $project_id"
        else
            print_error "No Google Cloud project configured"
            print_info "Set with: gcloud config set project YOUR-PROJECT-ID"
            return 1
        fi
    else
        print_success "Project configured: $project_id"
    fi

    # Check if Text-to-Speech API is enabled
    print_info "Checking if Text-to-Speech API is enabled..."
    local api_status
    api_status=$(gcloud services list --enabled --filter="name:texttospeech.googleapis.com" --format="value(name)" 2>/dev/null || echo "")

    if [ -z "$api_status" ]; then
        print_error "Text-to-Speech API is not enabled"
        print_info "Enable with: gcloud services enable texttospeech.googleapis.com"
        return 1
    fi

    print_success "Text-to-Speech API is enabled"

    # Test API access
    print_info "Testing Google Cloud TTS API access..."
    local access_token
    access_token=$(gcloud auth print-access-token 2>/dev/null || echo "")

    if [ -z "$access_token" ]; then
        print_error "Could not get access token"
        return 1
    fi

    if command -v curl >/dev/null 2>&1; then
        local response
        response=$(curl -s -w "%{http_code}" -o /dev/null \
            -H "Content-Type: application/json" \
            -H "Authorization: Bearer $access_token" \
            -H "X-Goog-User-Project: $project_id" \
            -d '{
                "input": {"text": "test"},
                "voice": {"languageCode": "en-US", "name": "en-US-Neural2-A"},
                "audioConfig": {"audioEncoding": "MP3"}
            }' \
            https://texttospeech.googleapis.com/v1/text:synthesize)

        if [ "$response" = "200" ]; then
            print_success "Google Cloud TTS API connection successful"
            return 0
        else
            print_error "Google Cloud TTS API connection failed (HTTP $response)"
            return 1
        fi
    else
        print_warning "curl not available, skipping API connection test"
        return 0
    fi
}

# Test Quacker app
test_app() {
    echo
    print_info "Testing Quacker application..."

    local app_path=""

    # Look for the app binary
    if [ -f "./easy-tts" ]; then
        app_path="./easy-tts"
    elif [ -f "./Quacker" ]; then
        app_path="./Quacker"
    elif [ -f "../easy-tts" ]; then
        app_path="../easy-tts"
    else
        print_error "Quacker app binary not found"
        print_info "Build with: go build ."
        return 1
    fi

    print_success "Found Quacker app: $app_path"

    # Check if app can run (just version check)
    if [ -x "$app_path" ]; then
        print_success "App binary is executable"
    else
        print_error "App binary is not executable"
        print_info "Fix with: chmod +x $app_path"
        return 1
    fi

    return 0
}

# Main test function
run_tests() {
    local openai_ok=false
    local google_ok=false
    local app_ok=false

    echo "Starting provider tests..."
    echo

    # Test app first
    if test_app; then
        app_ok=true
    fi

    # Test OpenAI
    if test_openai; then
        openai_ok=true
    fi

    # Test Google Cloud
    if test_google; then
        google_ok=true
    fi

    # Summary
    echo
    echo "📊 Test Summary"
    echo "==============="

    if $app_ok; then
        print_success "Quacker app: Ready"
    else
        print_error "Quacker app: Not ready"
    fi

    if $openai_ok; then
        print_success "OpenAI TTS: Ready"
    else
        print_error "OpenAI TTS: Not configured"
    fi

    if $google_ok; then
        print_success "Google Cloud TTS: Ready"
    else
        print_error "Google Cloud TTS: Not configured"
    fi

    echo

    if $openai_ok || $google_ok; then
        if $app_ok; then
            print_success "✨ You're ready to use Quacker!"
            echo
            print_info "Quick start:"
            print_info "1. Run: $([[ -f ./easy-tts ]] && echo "./easy-tts" || echo "./Quacker")"
            print_info "2. Select your provider from the dropdown"
            print_info "3. Enter text and click Submit"
        else
            print_warning "Providers are configured but app needs to be built"
        fi
    else
        print_error "No TTS providers are configured"
        echo
        print_info "To configure OpenAI:"
        print_info "  export OPENAI_API_KEY='your-key-here'"
        echo
        print_info "To configure Google Cloud:"
        print_info "  gcloud auth login"
        print_info "  gcloud config set project YOUR-PROJECT-ID"
        print_info "  gcloud services enable texttospeech.googleapis.com"
    fi

    echo
}

# Help function
show_help() {
    echo "Usage: $0 [option]"
    echo
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  --openai       Test only OpenAI configuration"
    echo "  --google       Test only Google Cloud configuration"
    echo "  --app          Test only Quacker app"
    echo "  (no option)    Test all providers and app"
    echo
    echo "Environment variables:"
    echo "  OPENAI_API_KEY        Your OpenAI API key"
    echo "  GOOGLE_CLOUD_PROJECT  Your Google Cloud project ID"
    echo "  GCP_PROJECT          Alternative Google Cloud project ID"
    echo
}

# Parse command line arguments
case "${1:-}" in
    -h|--help)
        show_help
        exit 0
        ;;
    --openai)
        test_openai
        exit $?
        ;;
    --google)
        test_google
        exit $?
        ;;
    --app)
        test_app
        exit $?
        ;;
    "")
        run_tests
        ;;
    *)
        echo "Unknown option: $1"
        show_help
        exit 1
        ;;
esac
