#!/bin/bash

# Force ARM64 mode explicitly
echo "Launching ARM64 bash shell..."
arch -arm64 /bin/bash <<'EOF'
echo "Shell architecture: $(uname -m)"

# Move to project directory
cd ./cmd/heliades || {
  echo "Could not find heliades directory."
  exit 1
}

# Show current Go arch
echo "Go architecture: $(go env GOARCH)"
echo "Rebuilding heliades binary for debugger..."

# Clean previous builds
go clean

# Build with debugger flags
go build -tags netgo,ledger -gcflags="all=-N -l" -o heliades

if [[ $? -ne 0 ]]; then
  echo "Build failed."
  exit 1
fi

echo "Build complete: heliades is ready for Delve or VSCode debug."
echo "Run it with: dlv exec ./heliades -- [your args here]"

# Keep session open for manual control
exec /bin/bash
EOF
