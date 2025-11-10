#!/usr/bin/env bash
set -euo pipefail

# Config
REPO="helios-network/helios-core"           # ex: "mon-org/mon-repo"
TAG="v0.0.254"
TITLE="Release ${TAG}"
NOTES="Initial release of the Helios Core"
BINARY_PATH="/Users/jeremyguyet/go/bin/heliades"  # chemin vers ton binaire
PRERELEASE=false

# Optionnel: target commit (branch)
TARGET="main"

# Vérifications
command -v gh >/dev/null || { echo "gh CLI introuvable. Installe https://cli.github.com/"; exit 1; }
[ -f "$BINARY_PATH" ] || { echo "Binaire introuvable: $BINARY_PATH"; exit 1; }

# Crée la release et upload l'asset
echo "Création de la release $TAG sur $REPO..."
if $PRERELEASE; then preflag="--prerelease"; else preflag=""; fi

gh release create "$TAG" "$BINARY_PATH" \
  --repo "$REPO" \
  --title "$TITLE" \
  --notes "$NOTES" \
  --target "$TARGET" $preflag

echo "Release créée et binaire uploadé."
sha256sum "$BINARY_PATH" | awk '{print $1}' > "${BINARY_PATH}.sha256"
echo "SHA256 écrit dans ${BINARY_PATH}.sha256"
