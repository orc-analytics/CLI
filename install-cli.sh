#!/bin/bash

set -e

REPO="orc-analytics/CLI"
INSTALL_NAME="orca"

# Disallow root user
if [ "$EUID" -eq 0 ]; then
  echo "Do not run this script as root. Please run as a regular user."
  exit 1
fi

# Detect OS type and architecture
detect_os() {
  UNAME="$(uname -s)"
  ARCH="$(uname -m)"
  case "$UNAME" in
    Darwin)
      OS="darwin"
      if [ "$ARCH" = "x86_64" ]; then
        ARCH_NAME="amd64"
      elif [ "$ARCH" = "arm64" ]; then
        ARCH_NAME="arm64"
      else
        echo "Unsupported Mac architecture: $ARCH"
        exit 1
      fi
      ;;
    Linux)
      OS="linux"
      if [ "$ARCH" = "x86_64" ]; then
        ARCH_NAME="amd64"
      elif [ "$ARCH" = "aarch64" ]; then
        ARCH_NAME="arm64"
      elif [ "$ARCH" = "i686" ] || [ "$ARCH" = "i386" ]; then
        ARCH_NAME="386"
      else
        echo "Unsupported Linux architecture: $ARCH"
        exit 1
      fi
      ;;
    MINGW*|MSYS*|CYGWIN*|Windows_NT)
      OS="windows"
      if [ "$ARCH" = "x86_64" ]; then
        ARCH_NAME="amd64"
      elif [ "$ARCH" = "i686" ] || [ "$ARCH" = "i386" ]; then
        ARCH_NAME="386"
      elif [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
        ARCH_NAME="arm64"
      else
        ARCH_NAME="amd64"  # Default for Windows
      fi
      ;;
    *)
      echo "Unsupported OS: $UNAME"
      exit 1
      ;;
  esac
}

# Get latest release version from GitHub API
get_latest_version() {
    echo "Fetching latest Orca CLI release..."
    LATEST_VERSION=$(curl -s https://api.github.com/repos/{$REPO}/releases/latest | \
      grep '"tag_name"' | \
      head -n 1 | \
      sed 's/.*"tag_name": "\([^"]*\)".*/\1/')

  echo "$LATEST_VERSION"
  
  if [ -z "$LATEST_VERSION" ] || [ "$LATEST_VERSION" = "null" ]; then
    echo "Failed to retrieve latest version"
    exit 1
  fi
  
  # Remove 'v' prefix if present
  LATEST_VERSION="${LATEST_VERSION#v}"
  echo "Latest version: $LATEST_VERSION"
}

# Download and extract the appropriate binary
download_binary() {
  ARCHIVE_NAME="CLI_${LATEST_VERSION}_${OS}_${ARCH_NAME}.tar.gz"
  DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${ARCHIVE_NAME}"
  TMP_DIR="$(mktemp -d)"
  TMP_ARCHIVE="$TMP_DIR/archive.tar.gz"
  
  echo "Downloading $DOWNLOAD_URL"
  if ! curl -fL "$DOWNLOAD_URL" -o "$TMP_ARCHIVE"; then
    echo "Failed to download binary"
    echo "URL attempted: $DOWNLOAD_URL"
    rm -rf "$TMP_DIR"
    exit 1
  fi
  
  echo "Extracting archive..."
  tar -xzf "$TMP_ARCHIVE" -C "$TMP_DIR"
  
  TMP_BINARY="$TMP_DIR/CLI"
  if [ ! -f "$TMP_BINARY" ]; then
    echo "Binary 'CLI' not found in archive"
    rm -rf "$TMP_DIR"
    exit 1
  fi
  
  chmod +x "$TMP_BINARY"
}

# Determine writable install directories
find_install_dirs() {
  # Create directories if they don't exist
  mkdir -p "$HOME/.local/bin" "$HOME/.local/share"
  
  SHARE_CANDIDATES=("$HOME/.local/share" "$HOME/share" "/usr/local/share")
  BIN_CANDIDATES=("$HOME/.local/bin" "$HOME/bin" "/usr/local/bin")

  for dir in "${SHARE_CANDIDATES[@]}"; do
    if [ -d "$dir" ] && [ -w "$dir" ]; then
      SHARE_DIR="$dir/orc_a"
      mkdir -p "$SHARE_DIR"
      break
    fi
  done

  for dir in "${BIN_CANDIDATES[@]}"; do
    if [ -d "$dir" ] && [ -w "$dir" ]; then
      BIN_DIR="$dir"
      break
    fi
  done

  if [ -z "$SHARE_DIR" ] || [ -z "$BIN_DIR" ]; then
    echo "No writable share/bin directory found. Please add one or run with elevated permissions."
    rm -rf "$TMP_DIR"
    exit 1
  fi
}

# Install binary and manage symlink
install_binary() {
  FINAL_BINARY="$SHARE_DIR/$INSTALL_NAME"
  SYMLINK_PATH="$BIN_DIR/$INSTALL_NAME"

  rm -f "$SYMLINK_PATH"

  mv "$TMP_BINARY" "$FINAL_BINARY"
  chmod +x "$FINAL_BINARY"
  ln -sf "$FINAL_BINARY" "$SYMLINK_PATH"
  
  # Clean up temp directory
  rm -rf "$TMP_DIR"

  echo ""
  echo "✅ Orca CLI installed to: $FINAL_BINARY"
  echo "✅ Symlink created at: $SYMLINK_PATH"
  echo "🔗 To get started, visit: https://github.com/orc-analytics/core#readme"
  
  # Verify installation
  if command -v "$INSTALL_NAME" &> /dev/null; then
    echo ""
    echo "Installation verified. Run '$INSTALL_NAME --version' to confirm."
  else
    echo ""
    echo "⚠️  Note: $BIN_DIR may not be in your PATH."
    echo "   Add it to your PATH by adding this line to your shell config:"
    echo "   export PATH=\"$BIN_DIR:\$PATH\""
  fi
}

# Run install steps
detect_os
get_latest_version
download_binary
find_install_dirs
install_binary
