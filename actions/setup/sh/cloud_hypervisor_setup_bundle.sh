#!/usr/bin/env bash
set +o histexpand

# cloud_hypervisor_setup_bundle.sh - Download, verify, and unpack AWF's
# cloud-hypervisor guest bundle for the requested AWF version.
#
# Outputs (GITHUB_OUTPUT):
#   binary_path, kernel_path, rootfs_path, supervisor_path
#   binary_sha256, kernel_sha256, rootfs_sha256, supervisor_sha256

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

bundle_root="${RUNNER_TEMP}/gh-aw/cloud-hypervisor/${version}"
extract_dir="${bundle_root}/bundle"
mkdir -p "${bundle_root}" "${extract_dir}"

echo "::group::Download cloud-hypervisor bundle (${version})"
curl -fsSL -o "${bundle_root}/${asset_name}" "${asset_base_url}/${asset_name}"
curl -fsSL -o "${bundle_root}/SHA256SUMS" "${asset_base_url}/SHA256SUMS"
curl -fsSL -o "${bundle_root}/manifest.json" "${asset_base_url}/manifest.json"
echo "downloaded release assets"
echo "::endgroup::"

echo "::group::Verify cloud-hypervisor bundle checksums"
for required_file in "${asset_name}" "manifest.json"; do
  if ! grep -Eq "(^|[[:space:]])(\./)?${required_file}$" "${bundle_root}/SHA256SUMS"; then
    echo "::error::${required_file} entry is missing in SHA256SUMS for ${version}"
    exit 1
  fi
  grep -E "(^|[[:space:]])(\./)?${required_file}$" "${bundle_root}/SHA256SUMS" | head -n1 | sha256sum -c -
done
echo "bundle and manifest checksums verified"
echo "::endgroup::"

echo "::group::Extract cloud-hypervisor bundle"
tar -xzf "${bundle_root}/${asset_name}" -C "${extract_dir}"
echo "bundle extracted to ${extract_dir}"
echo "::endgroup::"

manifest_path="${bundle_root}/manifest.json"
sha_file="${bundle_root}/SHA256SUMS"

pick_first_query() {
  local file="$1"
  shift
  local query
  for query in "$@"; do
    local value
    value="$(jq -r "${query} // empty" "${file}" 2>/dev/null | head -n1 || true)"
    if [[ -n "${value}" && "${value}" != "null" ]]; then
      echo "${value}"
      return 0
    fi
  done
  return 1
}

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
  for candidate in "${rel#./}" "$(basename "${rel#./}")" "${full#${bundle_root}/}" "${full#${extract_dir}/}"; do
    local sum
    sum="$(awk -v target="${candidate}" '{sub(/^\.\//, "", $2); if ($2==target) {print $1; exit}}' "${sha_file}")"
    if [[ -n "${sum}" ]]; then
      echo "${sum}"
      return 0
    fi
  done
  return 1
}

binary_rel="$(pick_first_query "${manifest_path}" \
  '.cloud_hypervisor.binary.path' \
  '.cloudHypervisor.binary.path' \
  '.artifacts.cloud_hypervisor.binary.path' \
  '.files[] | select((.role // .name // "") | test("binary"; "i")) | .path' || true)"
kernel_rel="$(pick_first_query "${manifest_path}" \
  '.cloud_hypervisor.kernel.path' \
  '.cloudHypervisor.kernel.path' \
  '.artifacts.cloud_hypervisor.kernel.path' \
  '.files[] | select((.role // .name // "") | test("kernel"; "i")) | .path' || true)"
rootfs_rel="$(pick_first_query "${manifest_path}" \
  '.cloud_hypervisor.rootfs.path' \
  '.cloudHypervisor.rootfs.path' \
  '.artifacts.cloud_hypervisor.rootfs.path' \
  '.files[] | select((.role // .name // "") | test("rootfs"; "i")) | .path' || true)"
supervisor_rel="$(pick_first_query "${manifest_path}" \
  '.cloud_hypervisor.supervisor.path' \
  '.cloudHypervisor.supervisor.path' \
  '.artifacts.cloud_hypervisor.supervisor.path' \
  '.files[] | select((.role // .name // "") | test("supervisor"; "i")) | .path' || true)"

if [[ -z "${binary_rel}" || -z "${kernel_rel}" || -z "${rootfs_rel}" || -z "${supervisor_rel}" ]]; then
  echo "::error::manifest.json is missing one or more required cloud-hypervisor artifact paths (binary/kernel/rootfs/supervisor)"
  exit 1
fi

binary_path="$(resolve_path "${binary_rel}" || true)"
kernel_path="$(resolve_path "${kernel_rel}" || true)"
rootfs_path="$(resolve_path "${rootfs_rel}" || true)"
supervisor_path="$(resolve_path "${supervisor_rel}" || true)"

if [[ -z "${binary_path}" || -z "${kernel_path}" || -z "${rootfs_path}" || -z "${supervisor_path}" ]]; then
  echo "::error::failed to resolve one or more cloud-hypervisor artifact files after extraction"
  exit 1
fi

binary_sha256="$(pick_first_query "${manifest_path}" \
  '.cloud_hypervisor.binary.sha256' \
  '.cloudHypervisor.binary.sha256' \
  '.artifacts.cloud_hypervisor.binary.sha256' || true)"
kernel_sha256="$(pick_first_query "${manifest_path}" \
  '.cloud_hypervisor.kernel.sha256' \
  '.cloudHypervisor.kernel.sha256' \
  '.artifacts.cloud_hypervisor.kernel.sha256' || true)"
rootfs_sha256="$(pick_first_query "${manifest_path}" \
  '.cloud_hypervisor.rootfs.sha256' \
  '.cloudHypervisor.rootfs.sha256' \
  '.artifacts.cloud_hypervisor.rootfs.sha256' || true)"
supervisor_sha256="$(pick_first_query "${manifest_path}" \
  '.cloud_hypervisor.supervisor.sha256' \
  '.cloudHypervisor.supervisor.sha256' \
  '.artifacts.cloud_hypervisor.supervisor.sha256' || true)"

binary_sha256="${binary_sha256:-$(lookup_sha256 "${binary_rel}" "${binary_path}" || true)}"
kernel_sha256="${kernel_sha256:-$(lookup_sha256 "${kernel_rel}" "${kernel_path}" || true)}"
rootfs_sha256="${rootfs_sha256:-$(lookup_sha256 "${rootfs_rel}" "${rootfs_path}" || true)}"
supervisor_sha256="${supervisor_sha256:-$(lookup_sha256 "${supervisor_rel}" "${supervisor_path}" || true)}"

if [[ -z "${binary_sha256}" || -z "${kernel_sha256}" || -z "${rootfs_sha256}" || -z "${supervisor_sha256}" ]]; then
  echo "::error::failed to resolve one or more cloud-hypervisor SHA256 digests from manifest.json/SHA256SUMS"
  exit 1
fi

if [[ -n "${GITHUB_OUTPUT:-}" ]]; then
  {
    echo "binary_path=${binary_path}"
    echo "kernel_path=${kernel_path}"
    echo "rootfs_path=${rootfs_path}"
    echo "supervisor_path=${supervisor_path}"
    echo "binary_sha256=${binary_sha256}"
    echo "kernel_sha256=${kernel_sha256}"
    echo "rootfs_sha256=${rootfs_sha256}"
    echo "supervisor_sha256=${supervisor_sha256}"
  } >> "${GITHUB_OUTPUT}"
fi
if [[ -n "${GITHUB_ENV:-}" ]]; then
  {
    echo "GH_AW_CLOUD_HYPERVISOR_BINARY=${binary_path}"
    echo "GH_AW_CLOUD_HYPERVISOR_KERNEL=${kernel_path}"
    echo "GH_AW_CLOUD_HYPERVISOR_ROOTFS=${rootfs_path}"
    echo "GH_AW_CLOUD_HYPERVISOR_SUPERVISOR=${supervisor_path}"
    echo "GH_AW_CLOUD_HYPERVISOR_BINARY_SHA256=${binary_sha256}"
    echo "GH_AW_CLOUD_HYPERVISOR_KERNEL_SHA256=${kernel_sha256}"
    echo "GH_AW_CLOUD_HYPERVISOR_ROOTFS_SHA256=${rootfs_sha256}"
    echo "GH_AW_CLOUD_HYPERVISOR_SUPERVISOR_SHA256=${supervisor_sha256}"
  } >> "${GITHUB_ENV}"
fi

echo "cloud-hypervisor bundle prepared"
