#!/usr/bin/env bash
# BranchProtection test suite.
#
# Test cases are discovered automatically from the filesystem:
#   test-[0-9][0-9]-<description>.yaml  →  BranchProtection CR
#   repos/repo-[0-9][0-9]-<description>.yaml  →  Repo CR (same index)
#
# The description comes from the filename slug after the numeric prefix,
# e.g. test-14-require-code-owners-and-last-push.yaml → "require-code-owners-and-last-push".
# All metadata (K8s names, GitHub repo name) is read directly from the YAML
# via yq — no hardcoded arrays. Adding a new test = drop a new file pair.
#
# Requires: kubectl, yq (mikefarah v4+)
#
# Usage:
#   ./run-tests.sh           # apply mode
#   ./run-tests.sh --cleanup # cleanup mode

set -euo pipefail

# ── Colours ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

# ── Paths ──────────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIGS_DIR="${SCRIPT_DIR}/../configs"
CRDS_DIR="${SCRIPT_DIR}/../../docs/crds"

# ── Constants ──────────────────────────────────────────────────────────────────
NAMESPACE="default"
TIMEOUT="210s"
GH_OWNER="krateoplatformops-test"
GH_SECRET="gh-token"
REPO_CONF_NAME="my-repo-config"
BP_CONF_NAME="my-branchprotection-config"
REPO_CONF_FILE="${CONFIGS_DIR}/sample-repo-config.yaml"
BP_CONF_FILE="${CONFIGS_DIR}/sample-branchprotection-config.yaml"

# ── Temp dir (removed on exit) ─────────────────────────────────────────────────
_TMP="$(mktemp -d)"
trap 'rm -rf "${_TMP}"' EXIT

# ── Test matrix — populated by discover_tests() ────────────────────────────────
BP_FILES=()        # path to test-XX-*.yaml
BP_K8S_NAMES=()    # .metadata.name from each BP file
BP_DESCRIPTIONS=() # filename slug:  test-03-enforce-admins-false → enforce-admins-false
REPO_FILES=()      # path to generated repo YAML in _TMP (one per BP file)
REPO_K8S_NAMES=()  # derived as repo-bp-tester-XX
REPO_GH_NAMES=()   # .spec.repo from each BP file  (= GitHub repo name)
N=0                # total test count; set by discover_tests()

# ── Logging ────────────────────────────────────────────────────────────────────
info()  { echo -e "${BLUE}[INFO]${NC}  $*"; }
ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
err()   { echo -e "${RED}[ERR]${NC}   $*" >&2; }
die()   { err "$*"; exit 1; }
step()  { echo -e "\n${BOLD}▶ $*${NC}"; }
indent(){ sed 's/^/        /'; }

# ── Tool check (needed before discovery) ───────────────────────────────────────
check_tools() {
  command -v kubectl &>/dev/null || die "kubectl not found in PATH."
  command -v yq      &>/dev/null || die "yq not found in PATH. Install: brew install yq"
  # Verify mikefarah yq v4+ syntax (dot-notation field access)
  echo 'name: ok' | yq '.name' &>/dev/null \
    || die "yq does not support '.field' syntax — mikefarah yq v4+ required."
}

# ── Test discovery ─────────────────────────────────────────────────────────────
# Scans SCRIPT_DIR for test-[0-9][0-9]-*.yaml files (sorted). For each one,
# generates a Repo CR on the fly using the constants already defined above
# (GH_OWNER, REPO_CONF_NAME, NAMESPACE) plus the GitHub repo name read from the
# BP file's .spec.repo field. No separate repo YAML files are needed.
discover_tests() {
  step "Discovering test cases"

  local bp_file base idx desc bp_name gh_repo repo_tmpfile
  while IFS= read -r bp_file; do
    base="$(basename "${bp_file}" .yaml)"   # e.g. test-03-enforce-admins-false
    idx="${base:5:2}"                        # e.g. 03  (chars 5-6 after "test-")
    desc="${base#test-??-}"                  # e.g. enforce-admins-false

    bp_name="$(yq '.metadata.name' "${bp_file}")"
    gh_repo="$(yq '.spec.repo'     "${bp_file}")"

    # Generate the Repo CR for this test into the temp directory.
    repo_tmpfile="${_TMP}/repo-${idx}.yaml"
    cat > "${repo_tmpfile}" <<EOF
apiVersion: github.ogen.krateo.io/v1alpha1
kind: Repo
metadata:
  name: repo-bp-tester-${idx}
  namespace: ${NAMESPACE}
  annotations:
    krateo.io/connector-verbose: "true"
spec:
  configurationRef:
    name: ${REPO_CONF_NAME}
    namespace: ${NAMESPACE}
  org: ${GH_OWNER}
  name: ${gh_repo}
  auto_init: true
  private: false
EOF

    BP_FILES+=("${bp_file}")
    BP_K8S_NAMES+=("${bp_name}")
    BP_DESCRIPTIONS+=("${desc}")
    REPO_FILES+=("${repo_tmpfile}")
    REPO_K8S_NAMES+=("repo-bp-tester-${idx}")
    REPO_GH_NAMES+=("${gh_repo}")
  done < <(find "${SCRIPT_DIR}" -maxdepth 1 -name 'test-[0-9][0-9]-*.yaml' | sort)

  N="${#BP_FILES[@]}"
  [[ "${N}" -gt 0 ]] \
    || die "No test files found matching test-[0-9][0-9]-*.yaml in ${SCRIPT_DIR}"

  ok "Discovered ${N} test cases"
  for i in "${!BP_DESCRIPTIONS[@]}"; do
    local num; printf -v num "%02d" $((i + 1))
    info "  ${num}  ${BP_DESCRIPTIONS[$i]}"
  done
}

# ── Cluster confirmation ────────────────────────────────────────────────────────
confirm_cluster() {
  local context server
  context="$(kubectl config current-context 2>/dev/null || echo 'unknown')"
  server="$(kubectl config view --minify \
    -o jsonpath='{.clusters[0].cluster.server}' 2>/dev/null || echo 'unknown')"

  echo ""
  echo -e "${BOLD}╔══════════════════════════════════════════════════════╗${NC}"
  echo -e "${BOLD}║           BranchProtection Test Suite                ║${NC}"
  echo -e "${BOLD}╠══════════════════════════════════════════════════════╣${NC}"
  printf  "${BOLD}║${NC}  %-18s ${YELLOW}%-32s${NC}${BOLD}║${NC}\n" "Context:"   "${context}"
  printf  "${BOLD}║${NC}  %-18s ${YELLOW}%-32s${NC}${BOLD}║${NC}\n" "Server:"    "${server}"
  printf  "${BOLD}║${NC}  %-18s ${YELLOW}%-32s${NC}${BOLD}║${NC}\n" "Namespace:" "${NAMESPACE}"
  printf  "${BOLD}║${NC}  %-18s ${YELLOW}%-32s${NC}${BOLD}║${NC}\n" "Org:"       "${GH_OWNER}"
  printf  "${BOLD}║${NC}  %-18s ${YELLOW}%-32s${NC}${BOLD}║${NC}\n" "Tests:"     "${N}"
  printf  "${BOLD}║${NC}  %-18s ${YELLOW}%-32s${NC}${BOLD}║${NC}\n" "Timeout:"   "${TIMEOUT} per resource"
  echo -e "${BOLD}╚══════════════════════════════════════════════════════╝${NC}"
  echo ""
  read -rp "Proceed on this cluster? [y/N] " _ans
  [[ "${_ans}" =~ ^[Yy]$ ]] || die "Aborted."
}

# ── Prerequisites ───────────────────────────────────────────────────────────────
check_prerequisites() {
  step "Checking prerequisites"

  local kver
  kver="$(kubectl version --client 2>/dev/null \
    | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo 'unknown')"
  ok "kubectl ${kver}"

  local yver
  yver="$(yq --version 2>/dev/null | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1 || echo 'unknown')"
  ok "yq ${yver}"

  kubectl cluster-info --request-timeout=10s &>/dev/null \
    || die "Cannot reach cluster. Check your kubeconfig / VPN."
  ok "Cluster is reachable"

  kubectl get secret "${GH_SECRET}" -n "${NAMESPACE}" &>/dev/null \
    || die "Secret '${GH_SECRET}' not found in namespace '${NAMESPACE}'." \
           "Create it: kubectl create secret generic ${GH_SECRET} --from-literal=token=<PAT> -n ${NAMESPACE}"
  ok "Secret '${GH_SECRET}' present"

  local crds=(
    "repoes.github.ogen.krateo.io:${CRDS_DIR}/repo-crd.yaml"
    "branchprotections.github.ogen.krateo.io:${CRDS_DIR}/branchprotection-crd.yaml"
    "repoconfigurations.github.ogen.krateo.io:${CRDS_DIR}/configuration-crds/repoconfiguration-crd.yaml"
    "branchprotectionconfigurations.github.ogen.krateo.io:${CRDS_DIR}/configuration-crds/branchprotectionconfiguration-crd.yaml"
  )
  local entry crd_name crd_file
  for entry in "${crds[@]}"; do
    crd_name="${entry%%:*}"
    crd_file="${entry##*:}"
    kubectl get crd "${crd_name}" &>/dev/null \
      || die "CRD '${crd_name}' not installed. Apply: kubectl apply -f ${crd_file}"
    ok "CRD ${crd_name}"
  done

  local f
  for f in "${BP_FILES[@]}"; do
    [[ -f "${f}" ]] || die "YAML file not found: ${f}"
  done
  ok "All ${N} BranchProtection YAML files present on disk (Repo CRs generated at runtime)"
}

# ── Config CR check / apply-if-missing ─────────────────────────────────────────
ensure_config_crs() {
  step "Ensuring configuration CRs"

  if kubectl get repoconfiguration "${REPO_CONF_NAME}" -n "${NAMESPACE}" &>/dev/null; then
    ok "RepoConfiguration '${REPO_CONF_NAME}' already present"
  else
    [[ -f "${REPO_CONF_FILE}" ]] || die "Config file not found: ${REPO_CONF_FILE}"
    info "Applying RepoConfiguration '${REPO_CONF_NAME}'..."
    kubectl apply -f "${REPO_CONF_FILE}" || die "Failed to apply ${REPO_CONF_FILE}"
    ok "RepoConfiguration '${REPO_CONF_NAME}' applied"
  fi

  if kubectl get branchprotectionconfiguration "${BP_CONF_NAME}" -n "${NAMESPACE}" &>/dev/null; then
    ok "BranchProtectionConfiguration '${BP_CONF_NAME}' already present"
  else
    [[ -f "${BP_CONF_FILE}" ]] || die "Config file not found: ${BP_CONF_FILE}"
    info "Applying BranchProtectionConfiguration '${BP_CONF_NAME}'..."
    kubectl apply -f "${BP_CONF_FILE}" || die "Failed to apply ${BP_CONF_FILE}"
    ok "BranchProtectionConfiguration '${BP_CONF_NAME}' applied"
  fi
}

# ── Parallel apply ─────────────────────────────────────────────────────────────
# apply_parallel <phase> <file_0> <k8s_name_0> [<file_1> <k8s_name_1> …]
# Writes ${_TMP}/<phase>/<name>.ok or .fail for each resource.
apply_parallel() {
  local phase="$1"; shift
  local subdir="${_TMP}/${phase}"
  mkdir -p "${subdir}"

  local pids=() names=()
  while [[ $# -ge 2 ]]; do
    local f="$1" name="$2"; shift 2
    names+=("${name}")
    (
      if kubectl apply -f "${f}" -n "${NAMESPACE}" >"${subdir}/${name}.out" 2>&1; then
        touch "${subdir}/${name}.ok"
      else
        touch "${subdir}/${name}.fail"
      fi
    ) &
    pids+=($!)
  done

  local pid; for pid in "${pids[@]}"; do wait "${pid}" || true; done

  local name failed=0
  for name in "${names[@]}"; do
    if [[ -f "${subdir}/${name}.ok" ]]; then
      ok "  applied    ${name}"
    else
      warn "  FAILED     ${name}"
      indent <"${subdir}/${name}.out"
      failed=1
    fi
  done
  return "${failed}"
}

# ── Parallel wait (condition=Ready or delete) ──────────────────────────────────
# wait_parallel <phase> <condition> <resource_type> <name_0> [<name_1> …]
# Writes ${_TMP}/<phase>/<name>.ok or .fail for each resource.
wait_parallel() {
  local phase="$1" condition="$2" rtype="$3"; shift 3
  local subdir="${_TMP}/${phase}"
  mkdir -p "${subdir}"

  local pids=()
  local name; for name in "$@"; do
    (
      if kubectl wait "${rtype}/${name}" \
           --for="${condition}" --timeout="${TIMEOUT}" \
           -n "${NAMESPACE}" >"${subdir}/${name}.out" 2>&1; then
        touch "${subdir}/${name}.ok"
      else
        touch "${subdir}/${name}.fail"
      fi
    ) &
    pids+=($!)
  done

  local pid; for pid in "${pids[@]}"; do wait "${pid}" || true; done

  for name in "$@"; do
    if [[ -f "${subdir}/${name}.ok" ]]; then
      ok "  ready      ${rtype}/${name}"
    else
      warn "  NOT READY  ${rtype}/${name} (timeout: ${TIMEOUT})"
      indent <"${subdir}/${name}.out"
    fi
  done
}

# ── Parallel delete (skip if not found) ────────────────────────────────────────
# delete_parallel <phase> <resource_type> <name_0> [<name_1> …]
# Writes ${_TMP}/<phase>/<name>.deleted, .notfound, or .fail.
delete_parallel() {
  local phase="$1" rtype="$2"; shift 2
  local subdir="${_TMP}/${phase}"
  mkdir -p "${subdir}"

  local pids=()
  local name; for name in "$@"; do
    (
      if ! kubectl get "${rtype}/${name}" -n "${NAMESPACE}" &>/dev/null; then
        touch "${subdir}/${name}.notfound"
      elif kubectl delete "${rtype}/${name}" -n "${NAMESPACE}" \
             >"${subdir}/${name}.out" 2>&1; then
        touch "${subdir}/${name}.deleted"
      else
        touch "${subdir}/${name}.fail"
      fi
    ) &
    pids+=($!)
  done

  local pid; for pid in "${pids[@]}"; do wait "${pid}" || true; done

  for name in "$@"; do
    if   [[ -f "${subdir}/${name}.deleted"  ]]; then ok   "  deleted    ${rtype}/${name}"
    elif [[ -f "${subdir}/${name}.notfound" ]]; then info "  not found  ${rtype}/${name} — skipped"
    else warn "  FAILED     ${rtype}/${name}"; indent <"${subdir}/${name}.out"
    fi
  done
}

# ── Summary table ──────────────────────────────────────────────────────────────
print_summary() {
  local repo_wait_dir="${_TMP}/repo-wait"
  local bp_apply_dir="${_TMP}/bp-apply"
  local bp_wait_dir="${_TMP}/bp-wait"

  echo ""
  echo -e "${BOLD}╔══════════════════════════════════════════════════════════════════╗${NC}"
  printf  "${BOLD}║  %-4s  %-36s  %-8s  %-8s  ║${NC}\n" "#" "DESCRIPTION" "REPO" "BP"
  echo -e "${BOLD}╠══════════════════════════════════════════════════════════════════╣${NC}"

  local pass=0 fail=0 i rname bname num repo_col bp_col
  for i in "${!REPO_K8S_NAMES[@]}"; do
    rname="${REPO_K8S_NAMES[$i]}"
    bname="${BP_K8S_NAMES[$i]}"
    printf -v num "%02d" $((i + 1))

    if   [[ -f "${repo_wait_dir}/${rname}.ok"   ]]; then repo_col="${GREEN}Ready${NC}"
    elif [[ -f "${repo_wait_dir}/${rname}.fail"  ]]; then repo_col="${RED}FAILED${NC}"
    else                                                  repo_col="${YELLOW}?${NC}"
    fi

    if   [[ -f "${bp_wait_dir}/${bname}.ok"    ]]; then bp_col="${GREEN}Ready${NC}";    (( ++pass ))
    elif [[ -f "${bp_wait_dir}/${bname}.fail"  ]]; then bp_col="${RED}FAILED${NC}";     (( ++fail ))
    elif [[ -f "${bp_apply_dir}/${bname}.fail" ]]; then bp_col="${RED}APPLY ERR${NC}";  (( ++fail ))
    else                                               bp_col="${DIM}SKIPPED${NC}";   (( ++fail ))
    fi

    printf "${BOLD}║${NC}  %02d   %-36s  " $((i + 1)) "${BP_DESCRIPTIONS[$i]}"
    echo -en "${repo_col}        "
    echo -e  "${bp_col}  ${BOLD}║${NC}"
  done

  echo -e "${BOLD}╠══════════════════════════════════════════════════════════════════╣${NC}"
  echo -e "${BOLD}║  Total: ${GREEN}${pass} passed${NC}${BOLD},  ${RED}${fail} failed/skipped${NC}${BOLD}                                 ║${NC}"
  echo -e "${BOLD}╚══════════════════════════════════════════════════════════════════╝${NC}"
  echo ""
}

# ── Apply mode ─────────────────────────────────────────────────────────────────
run_apply() {

  # Step 1 — Apply all Repo CRs in parallel
  step "Step 1/4 — Applying ${N} Repo CRs in parallel"
  local repo_apply_args=()
  local i; for i in "${!REPO_FILES[@]}"; do
    repo_apply_args+=("${REPO_FILES[$i]}" "${REPO_K8S_NAMES[$i]}")
  done
  apply_parallel "repo-apply" "${repo_apply_args[@]}" || true

  # Step 2 — Wait for all Repos to become Ready
  step "Step 2/4 — Waiting for ${N} Repo CRs (timeout: ${TIMEOUT} each)"
  wait_parallel "repo-wait" "condition=Ready" "repo" "${REPO_K8S_NAMES[@]}"

  # Build BP apply list — only for repos that became Ready
  local bp_apply_args=() bp_wait_names=() skipped=0
  for i in "${!REPO_K8S_NAMES[@]}"; do
    if [[ -f "${_TMP}/repo-wait/${REPO_K8S_NAMES[$i]}.ok" ]]; then
      bp_apply_args+=("${BP_FILES[$i]}" "${BP_K8S_NAMES[$i]}")
      bp_wait_names+=("${BP_K8S_NAMES[$i]}")
    else
      warn "Skipping BranchProtection '${BP_DESCRIPTIONS[$i]}' — repo did not become Ready"
      (( ++skipped )) || true
    fi
  done
  [[ ${skipped} -gt 0 ]] && warn "${skipped} BranchProtection CR(s) skipped due to repo failures"

  if [[ ${#bp_wait_names[@]} -eq 0 ]]; then
    err "All Repo CRs failed — no BranchProtection CRs to apply."
    print_summary; exit 1
  fi

  # Step 3 — Apply BranchProtection CRs for ready repos
  step "Step 3/4 — Applying ${#bp_wait_names[@]} BranchProtection CRs in parallel"
  apply_parallel "bp-apply" "${bp_apply_args[@]}" || true

  # Narrow to BPs that were actually applied successfully
  local bp_ready_names=()
  local name; for name in "${bp_wait_names[@]}"; do
    [[ -f "${_TMP}/bp-apply/${name}.ok" ]] && bp_ready_names+=("${name}") || true
  done

  # Step 4 — Wait for BranchProtections to become Ready
  if [[ ${#bp_ready_names[@]} -gt 0 ]]; then
    step "Step 4/4 — Waiting for ${#bp_ready_names[@]} BranchProtection CRs (timeout: ${TIMEOUT} each)"
    wait_parallel "bp-wait" "condition=Ready" "branchprotection" "${bp_ready_names[@]}"
  else
    warn "No BranchProtection CRs were applied successfully — nothing to wait on"
  fi

  print_summary
}

# ── Cleanup mode ───────────────────────────────────────────────────────────────
run_cleanup() {
  echo ""
  echo -e "${RED}${BOLD}╔══════════════════════════════════════════════════════════════════╗${NC}"
  echo -e "${RED}${BOLD}║  WARNING — the following GitHub repositories will be DELETED:   ║${NC}"
  echo -e "${RED}${BOLD}╠══════════════════════════════════════════════════════════════════╣${NC}"
  local name; for name in "${REPO_GH_NAMES[@]}"; do
    printf "${RED}${BOLD}║${NC}    ${RED}%-64s${NC}${RED}${BOLD}║${NC}\n" "${GH_OWNER}/${name}"
  done
  echo -e "${RED}${BOLD}╚══════════════════════════════════════════════════════════════════╝${NC}"
  echo ""
  echo -e "${YELLOW}This action is irreversible. Type the org name to confirm:${NC}"
  read -rp "  Org [${GH_OWNER}]: " _conf
  [[ "${_conf}" == "${GH_OWNER}" ]] || die "Org name mismatch — aborted."

  # Step 1 — Delete BranchProtection CRs (controller calls GitHub DELETE)
  step "Step 1/4 — Deleting BranchProtection CRs in parallel"
  delete_parallel "bp-delete" "branchprotection" "${BP_K8S_NAMES[@]}"

  # Step 2 — Wait for finalizers to complete
  step "Step 2/4 — Waiting for BranchProtection CRs to be deleted (timeout: ${TIMEOUT} each)"
  local existing_bps=()
  for name in "${BP_K8S_NAMES[@]}"; do
    kubectl get "branchprotection/${name}" -n "${NAMESPACE}" &>/dev/null \
      && existing_bps+=("${name}") || true
  done
  if [[ ${#existing_bps[@]} -gt 0 ]]; then
    wait_parallel "bp-delete-wait" "delete" "branchprotection" "${existing_bps[@]}"
  else
    info "No BranchProtection CRs remain — nothing to wait on"
  fi

  # Step 3 — Delete Repo CRs (controller calls GitHub API to delete repos)
  step "Step 3/4 — Deleting Repo CRs in parallel"
  delete_parallel "repo-delete" "repo" "${REPO_K8S_NAMES[@]}"

  # Step 4 — Wait for finalizers to complete
  step "Step 4/4 — Waiting for Repo CRs to be deleted (timeout: ${TIMEOUT} each)"
  local existing_repos=()
  for name in "${REPO_K8S_NAMES[@]}"; do
    kubectl get "repo/${name}" -n "${NAMESPACE}" &>/dev/null \
      && existing_repos+=("${name}") || true
  done
  if [[ ${#existing_repos[@]} -gt 0 ]]; then
    wait_parallel "repo-delete-wait" "delete" "repo" "${existing_repos[@]}"
  else
    info "No Repo CRs remain — nothing to wait on"
  fi

  echo ""; ok "Cleanup complete. All test CRs deleted."; echo ""
}

# ── Main ───────────────────────────────────────────────────────────────────────
MODE="${1:-apply}"
[[ "${MODE}" == "--cleanup" ]] && MODE="cleanup"
[[ "${MODE}" == "apply" || "${MODE}" == "cleanup" ]] \
  || die "Usage: $(basename "$0") [--cleanup]"

echo -e "\n${BOLD}BranchProtection Test Suite${NC} — mode: ${YELLOW}${MODE}${NC}"

check_tools       # verify kubectl + yq before anything else
discover_tests    # populate arrays from filesystem
confirm_cluster   # show N (now known) and ask for confirmation
check_prerequisites
ensure_config_crs

if [[ "${MODE}" == "apply" ]]; then
  run_apply
else
  run_cleanup
fi
