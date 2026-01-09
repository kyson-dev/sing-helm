#!/bin/bash
set -e

# update-homebrew.sh
# Usage: ./update-homebrew.sh <version>
# Env: GITHUB_TOKEN (repo scope token for homebrew tap)

VERSION=$1
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

echo "Updating Homebrew formula for version ${VERSION}..."

# Calculate SHA256 checksums
ARM64_SHA=$(sha256sum bin/sing-helm-darwin-arm64 | awk '{print $1}')
AMD64_SHA=$(sha256sum bin/sing-helm-darwin-amd64 | awk '{print $1}')

echo "Calculated SHA256:"
echo "  ARM64: ${ARM64_SHA}"
echo "  AMD64: ${AMD64_SHA}"

# Prepare temp directory
WORK_DIR=$(mktemp -d)
echo "Working in ${WORK_DIR}..."

# Clone homebrew tap repo
git clone https://github.com/kyson-dev/homebrew-sing-helm.git "${WORK_DIR}"
cd "${WORK_DIR}"

# Generate new formula content
cat > Formula/sing-helm.rb <<EOF
class SingHelm < Formula
  desc "Lightweight sing-box configuration manager and proxy client"
  homepage "https://github.com/kyson-dev/sing-helm"
  version "${VERSION}"
  
  if Hardware::CPU.arm?
    url "https://github.com/kyson-dev/sing-helm/releases/download/v${VERSION}/sing-helm-darwin-arm64"
    sha256 "${ARM64_SHA}"
  else
    url "https://github.com/kyson-dev/sing-helm/releases/download/v${VERSION}/sing-helm-darwin-amd64"
    sha256 "${AMD64_SHA}"
  end

  def install
    bin.install "sing-helm-darwin-arm64" => "sing-helm" if Hardware::CPU.arm?
    bin.install "sing-helm-darwin-amd64" => "sing-helm" if Hardware::CPU.intel?
  end

  def caveats
    <<~EOS
      To start sing-helm as a system service:
        sudo sing-helm autostart on
      
      To run sing-helm manually:
        sudo sing-helm run
    EOS
  end

  test do
    system "#{bin}/sing-helm", "version"
  end
end
EOF

# Check if there are changes
if git diff --quiet Formula/sing-helm.rb; then
    echo "No changes detected in formula."
    exit 0
fi

# Configure git
git config user.name "GitHub Actions"
git config user.email "actions@github.com"

# Commit and push
git add Formula/sing-helm.rb
git commit -m "Update sing-helm to v${VERSION}"
git push "https://x-access-token:${GITHUB_TOKEN}@github.com/kyson-dev/homebrew-sing-helm.git" main

echo "✅ Homebrew formula updated successfully!"
