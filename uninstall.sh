#!/bin/sh
set -e

BINARY="kubie-go"
INSTALL_DIR="/usr/local/bin"

remove() {
  if [ -f "${INSTALL_DIR}/$1" ] || [ -L "${INSTALL_DIR}/$1" ]; then
    if [ -w "$INSTALL_DIR" ]; then
      rm -f "${INSTALL_DIR}/$1"
    else
      sudo rm -f "${INSTALL_DIR}/$1"
    fi
    echo "Removed ${INSTALL_DIR}/$1"
  fi
}

remove "$BINARY"
remove "kubie"

echo "Done! kubie-go has been uninstalled."
