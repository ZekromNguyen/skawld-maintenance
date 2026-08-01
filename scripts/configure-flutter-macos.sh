#!/usr/bin/env bash
set -euo pipefail

app_identifier="com.skawld.skawldMaintenanceMobile"
entitlements=(
  "macos/Runner/DebugProfile.entitlements"
  "macos/Runner/Release.entitlements"
)

for entitlement in "${entitlements[@]}"; do
  if [[ ! -f "${entitlement}" ]]; then
    echo "Generated Flutter entitlement file does not exist: ${entitlement}" >&2
    exit 1
  fi

  /usr/libexec/PlistBuddy \
    -c "Set :com.apple.security.network.client true" \
    "${entitlement}" 2>/dev/null ||
    /usr/libexec/PlistBuddy \
      -c "Add :com.apple.security.network.client bool true" \
      "${entitlement}"

  /usr/libexec/PlistBuddy \
    -c "Set :com.apple.security.network.server true" \
    "${entitlement}" 2>/dev/null ||
    /usr/libexec/PlistBuddy \
      -c "Add :com.apple.security.network.server bool true" \
      "${entitlement}"

  if /usr/libexec/PlistBuddy \
    -c "Print :keychain-access-groups" \
    "${entitlement}" >/dev/null 2>&1; then
    /usr/libexec/PlistBuddy -c "Delete :keychain-access-groups" "${entitlement}"
  fi
  /usr/libexec/PlistBuddy -c "Add :keychain-access-groups array" "${entitlement}"
  /usr/libexec/PlistBuddy \
    -c 'Add :keychain-access-groups:0 string $(AppIdentifierPrefix)com.skawld.skawldMaintenanceMobile' \
    "${entitlement}"
done

echo "Configured macOS OIDC loopback, network, and Keychain entitlements for ${app_identifier}"
