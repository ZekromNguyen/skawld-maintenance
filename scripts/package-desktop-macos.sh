#!/usr/bin/env bash
set -euo pipefail

version="${1:-dev}"
app_path="${2:-build/macos/Build/Products/Release/skawld_maintenance_mobile.app}"
output_directory="${3:-dist/desktop}"
script_directory="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
configuration_template="${script_directory}/../deployments/desktop/skawld-config.example.json"

if [[ ! -d "${app_path}" ]]; then
  echo "macOS application bundle does not exist: ${app_path}" >&2
  exit 1
fi
if [[ ! -f "${configuration_template}" ]]; then
  echo "Desktop configuration template does not exist: ${configuration_template}" >&2
  exit 1
fi

mkdir -p "${output_directory}"
dmg_path="${output_directory}/skawld-maintenance-${version}-macos.dmg"
zip_path="${output_directory}/skawld-maintenance-${version}-macos.zip"
rm -f "${dmg_path}" "${zip_path}" "${dmg_path}.sha256" "${zip_path}.sha256"
staging_directory="$(mktemp -d)"
trap 'rm -r -- "${staging_directory}"' EXIT
cp -R "${app_path}" "${staging_directory}/"
cp "${configuration_template}" "${staging_directory}/skawld-config.example.json"

hdiutil create \
  -volname "Skawld Maintenance" \
  -srcfolder "${staging_directory}" \
  -ov \
  -format UDZO \
  "${dmg_path}"
ditto -c -k --sequesterRsrc "${staging_directory}" "${zip_path}"

(
  cd "${output_directory}"
  shasum -a 256 "$(basename "${dmg_path}")" > "$(basename "${dmg_path}").sha256"
  shasum -a 256 "$(basename "${zip_path}")" > "$(basename "${zip_path}").sha256"
)

echo "Created ${dmg_path} and ${zip_path}"
