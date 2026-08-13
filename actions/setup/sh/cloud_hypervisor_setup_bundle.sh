#!/usr/bin/env bash
set +o histexpand

# cloud_hypervisor_setup_bundle.sh - Download, verify, and unpack AWF's
# cloud-hypervisor guest bundle for the requested AWF version.
#
# Outputs (GITHUB_OUTPUT):
#   binary_path, virtiofsd_path, kernel_path, rootfs_path, supervisor_path
#   binary_sha256, virtiofsd_sha256, kernel_sha256, rootfs_sha256,
#   supervisor_sha256

set -euo pipefail

if [[ -z "${GH_AW_AWF_VERSION:-}" ]]; then
  echo "::error::GH_AW_AWF_VERSION is required"
  exit 1
fi

version="${GH_AW_AWF_VERSION}"
if [[ "${version}" != v* ]]; then
  version="v${version}"
fi

asset_base_url="https://github.com/github/gh-aw-firewall/releases/download/${version}"
asset_name="cloud-hypervisor-test-x86_64.tar.gz"
checksums_name="cloud-hypervisor-test-x86_64.SHA256SUMS"
manifest_name="cloud-hypervisor-test-x86_64.manifest.json"

bundle_root="${RUNNER_TEMP}/gh-aw/cloud-hypervisor/${version}"
extract_dir="${bundle_root}/bundle"
mkdir -p "${bundle_root}" "${extract_dir}"

echo "::group::Download cloud-hypervisor bundle (${version})"
curl -fsSL -o "${bundle_root}/${asset_name}" "${asset_base_url}/${asset_name}"
curl -fsSL -o "${bundle_root}/${checksums_name}" "${asset_base_url}/${checksums_name}"
curl -fsSL -o "${bundle_root}/${manifest_name}" "${asset_base_url}/${manifest_name}"
echo "downloaded release assets"
echo "::endgroup::"

echo "::group::Extract cloud-hypervisor bundle"
tar -xzf "${bundle_root}/${asset_name}" -C "${extract_dir}"
echo "bundle extracted to ${extract_dir}"
echo "::endgroup::"

sha_file="${bundle_root}/${checksums_name}"

resolve_path() {
  local rel="$1"
  if [[ -z "${rel}" ]]; then
    return 1
  fi

  local cleaned="${rel#./}"
  local candidate
  for candidate in \
    "${extract_dir}/${cleaned}" \
    "${bundle_root}/${cleaned}"; do
    if [[ -f "${candidate}" ]]; then
      realpath "${candidate}"
      return 0
    fi
  done

  local found
  found="$(find "${extract_dir}" -type f -name "$(basename "${cleaned}")" | head -n1 || true)"
  if [[ -n "${found}" ]]; then
    realpath "${found}"
    return 0
  fi

  return 1
}

lookup_sha256() {
  local rel="$1"
  local full="$2"
  local candidate
  for candidate in "${rel#./}" "$(basename "${rel#./}")" "${full#"${bundle_root}"/}" "${full#"${extract_dir}"/}"; do
    local sum
    sum="$(awk -v target="${candidate}" '{sub(/^\.\//, "", $2); if ($2==target) {print $1; exit}}' "${sha_file}")"
    if [[ -n "${sum}" ]]; then
      echo "${sum}"
      return 0
    fi
  done
  return 1
}

verify_sha256() {
  local expected="$1"
  local file="$2"
  local actual
  actual="$(sha256sum "${file}" | awk '{print $1}')"
  if [[ "${actual}" != "${expected}" ]]; then
    echo "::error::checksum verification failed for ${file}"
    exit 1
  fi
}

# Artifact names are fixed by the gh-aw-firewall cloud-hypervisor release contract.
binary_rel="cloud-hypervisor"
kernel_rel="vmlinux.bin"
rootfs_rel="rootfs.ext4"
supervisor_rel="awf-supervisor"
virtiofsd_rel="virtiofsd"

binary_path="$(resolve_path "${binary_rel}" || true)"
kernel_path="$(resolve_path "${kernel_rel}" || true)"
rootfs_path="$(resolve_path "${rootfs_rel}" || true)"
supervisor_path="$(resolve_path "${supervisor_rel}" || true)"
virtiofsd_path="$(resolve_path "${virtiofsd_rel}" || true)"

if [[ -z "${binary_path}" || -z "${kernel_path}" || -z "${rootfs_path}" || -z "${supervisor_path}" || -z "${virtiofsd_path}" ]]; then
  echo "::error::failed to resolve one or more cloud-hypervisor artifact files after extraction"
  exit 1
fi

if [[ "$(dirname "${binary_path}")" != "$(dirname "${virtiofsd_path}")" ]]; then
  echo "::error::virtiofsd must be colocated with the cloud-hypervisor binary"
  exit 1
fi

binary_sha256="$(lookup_sha256 "${binary_rel}" "${binary_path}" || true)"
kernel_sha256="$(lookup_sha256 "${kernel_rel}" "${kernel_path}" || true)"
rootfs_sha256="$(lookup_sha256 "${rootfs_rel}" "${rootfs_path}" || true)"
supervisor_sha256="$(lookup_sha256 "${supervisor_rel}" "${supervisor_path}" || true)"
virtiofsd_sha256="$(lookup_sha256 "${virtiofsd_rel}" "${virtiofsd_path}" || true)"

if [[ -z "${binary_sha256}" || -z "${kernel_sha256}" || -z "${rootfs_sha256}" || -z "${supervisor_sha256}" || -z "${virtiofsd_sha256}" ]]; then
  echo "::error::failed to resolve one or more cloud-hypervisor SHA256 digests from ${checksums_name}"
  exit 1
fi

manifest_path="${bundle_root}/${manifest_name}"
if ! jq -e '
  .schemaVersion == 1
  and .architecture == "x86_64"
  and (.cloudHypervisor.version | type == "string" and length > 0)
  and (.cloudHypervisor.binarySha256 | type == "string" and test("^[0-9A-Fa-f]{64}$"))
  and (.virtiofsd.version | type == "string" and length > 0)
  and (.virtiofsd.binarySha256 | type == "string" and test("^[0-9A-Fa-f]{64}$"))
' "${manifest_path}" >/dev/null; then
  echo "::error::${manifest_name} does not match the cloud-hypervisor release bundle contract"
  exit 1
fi

manifest_binary_sha256="$(jq -r '.cloudHypervisor.binarySha256 | ascii_downcase' "${manifest_path}")"
manifest_virtiofsd_sha256="$(jq -r '.virtiofsd.binarySha256 | ascii_downcase' "${manifest_path}")"
if [[ "${manifest_binary_sha256}" != "${binary_sha256,,}" || "${manifest_virtiofsd_sha256}" != "${virtiofsd_sha256,,}" ]]; then
  echo "::error::${manifest_name} digest fields do not match ${checksums_name}"
  exit 1
fi

echo "::group::Verify cloud-hypervisor bundle checksums"
verify_sha256 "${binary_sha256}" "${binary_path}"
verify_sha256 "${kernel_sha256}" "${kernel_path}"
verify_sha256 "${rootfs_sha256}" "${rootfs_path}"
verify_sha256 "${supervisor_sha256}" "${supervisor_path}"
verify_sha256 "${virtiofsd_sha256}" "${virtiofsd_path}"
chmod 0755 "${binary_path}" "${supervisor_path}" "${virtiofsd_path}"
echo "bundle checksums verified"
echo "::endgroup::"

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  {
    echo "binary_path=${binary_path}"
    echo "kernel_path=${kernel_path}"
    echo "rootfs_path=${rootfs_path}"
    echo "supervisor_path=${supervisor_path}"
    echo "virtiofsd_path=${virtiofsd_path}"
    echo "binary_sha256=${binary_sha256}"
    echo "kernel_sha256=${kernel_sha256}"
    echo "rootfs_sha256=${rootfs_sha256}"
    echo "supervisor_sha256=${supervisor_sha256}"
    echo "virtiofsd_sha256=${virtiofsd_sha256}"
  } >> "${GITHUB_OUTPUT}"
fi
if [[ -n "${GITHUB_ENV:-}" ]]; then
  {
    echo "GH_AW_CLOUD_HYPERVISOR_BINARY=${binary_path}"
    echo "GH_AW_CLOUD_HYPERVISOR_KERNEL=${kernel_path}"
    echo "GH_AW_CLOUD_HYPERVISOR_ROOTFS=${rootfs_path}"
    echo "GH_AW_CLOUD_HYPERVISOR_SUPERVISOR=${supervisor_path}"
    echo "GH_AW_CLOUD_HYPERVISOR_VIRTIOFSD=${virtiofsd_path}"
    echo "GH_AW_CLOUD_HYPERVISOR_BINARY_SHA256=${binary_sha256}"
    echo "GH_AW_CLOUD_HYPERVISOR_KERNEL_SHA256=${kernel_sha256}"
    echo "GH_AW_CLOUD_HYPERVISOR_ROOTFS_SHA256=${rootfs_sha256}"
    echo "GH_AW_CLOUD_HYPERVISOR_SUPERVISOR_SHA256=${supervisor_sha256}"
    echo "GH_AW_CLOUD_HYPERVISOR_VIRTIOFSD_SHA256=${virtiofsd_sha256}"
  } >> "${GITHUB_ENV}"
fi

echo "cloud-hypervisor bundle prepared"
