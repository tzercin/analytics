#!/usr/bin/env bash
#
# github-ruleset.sh - inspect, diff, and reconcile the branch ruleset that
# protects this repository's default branch.
#
# Read-only by default. Every mutation needs both its own subcommand and the
# confirmation token printed by an immediately preceding dry run.
#
# Requires: gh (authenticated), jq, shasum.

set -euo pipefail

REPO="${RULESET_REPO:-tzercin/analytics}"
BRANCH="${RULESET_BRANCH:-master}"
CONFIG="${RULESET_CONFIG:-.github/rulesets/protect-master.json}"
NAME="protect-master"
API_VERSION="2026-03-10"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

usage() {
  cat >&2 <<'EOF'
Usage: scripts/github-ruleset.sh <command> [options]

Commands:
  validate            Offline schema/policy checks on the config file.
  plan                (default) Read-only inspection and diff against GitHub.
  apply               Create or update the ruleset.   Needs --confirm TOKEN.
  disable             Set enforcement to disabled.    Needs --confirm TOKEN.
  delete              Delete the ruleset (final).     Needs --confirm TOKEN.

Options:
  --enforcement VALUE   Override desired enforcement: active | disabled.
  --confirm TOKEN       Token from the immediately preceding dry run.

Environment:
  RULESET_REPO (default tzercin/analytics)
  RULESET_BRANCH (default master)
  RULESET_CONFIG (default .github/rulesets/protect-master.json)

Without --confirm, apply/disable/delete only print a plan. They never mutate.
EOF
}

die() { printf 'error: %s\n' "$*" >&2; exit 1; }
note() { printf '%s\n' "$*" >&2; }

api() {
  gh api -H "Accept: application/vnd.github+json" \
         -H "X-GitHub-Api-Version: ${API_VERSION}" "$@"
}

hash_file() { shasum -a 256 "$1" | cut -d' ' -f1; }

# GitHub echoes neutral defaults that were never sent. Strip only those, sort
# every set-like array, and keep just the declarative payload fields so that a
# diff reflects policy rather than serialization.
# $v below is a jq binding, not a shell variable.
# shellcheck disable=SC2016
JQ_NORMALIZE='
  {name, target, enforcement, bypass_actors, conditions, rules}
  | .bypass_actors = ((.bypass_actors // [])
      | map({actor_id, actor_type, bypass_mode}) | sort_by(tojson))
  | .conditions.ref_name.include = ((.conditions.ref_name.include // []) | sort)
  | .conditions.ref_name.exclude = ((.conditions.ref_name.exclude // []) | sort)
  | .rules = ((.rules // [])
      | map(if .type == "pull_request" then
              .parameters |= (del(.dismissal_restriction,
                                  .ignore_approvals_from_contributors,
                                  .required_reviewers)
                              | .allowed_merge_methods = ((.allowed_merge_methods // []) | sort)
                              # GitHub enables this by default on new and existing
                              # rulesets, so an omission on either side means true.
                              # Test for null explicitly: jq "//" also swallows
                              # a genuine false, which would hide a real change.
                              | .require_extra_approval_for_unattributed_changes =
                                  (.require_extra_approval_for_unattributed_changes as $v
                                   | if $v == null then true else $v end))
            else . end)
      | sort_by(.type))
'

# Refuse to normalize away a server-side setting that is NOT neutral, because
# stripping it would silently overwrite real policy on the next apply. Settings
# this config declares explicitly are compared in the diff instead, so they are
# deliberately absent from this list.
JQ_NON_NEUTRAL='
  [ (.rules // [])[] | select(.type == "pull_request") | .parameters
    | ( ((.dismissal_restriction.enabled // false) == true),
        (((.dismissal_restriction.allowed_actors // []) | length) > 0),
        ((.ignore_approvals_from_contributors // false) == true),
        (((.required_reviewers // []) | length) > 0) ) ]
  | any
'

# The merge methods a ruleset would allow. Kept as a named program so the test
# suite asserts the exact path the preflight check reads.
JQ_MERGE_METHODS='.rules[] | select(.type == "pull_request")
                           | .parameters.allowed_merge_methods[]'

# $name and $branch below are jq variables bound with --arg, not shell variables.
# shellcheck disable=SC2016
JQ_POLICY_ERRORS='
  [ (if .name != $name then "name must be \"" + $name + "\"" else empty end),
    (if .target != "branch" then "target must be \"branch\"" else empty end),
    (if (.enforcement == "active" or .enforcement == "disabled") | not
       then "enforcement must be active or disabled (evaluate is Enterprise-only)" else empty end),
    (if (.conditions.ref_name.include // []) != ["refs/heads/" + $branch]
       then "conditions.ref_name.include must be exactly [\"refs/heads/" + $branch + "\"]" else empty end),
    (if ((.conditions.ref_name.exclude // null) | type) != "array"
       then "conditions.ref_name.exclude must be an explicit array" else empty end),
    (if ((.conditions.ref_name.exclude // []) | length) != 0
       then "conditions.ref_name.exclude must be empty" else empty end),
    (if ((.bypass_actors // []) | length) != 1
       then "bypass_actors must hold exactly the admin recovery entry" else empty end),
    ((.bypass_actors // [])[]
       | (if .actor_type != "RepositoryRole" then "bypass actor_type must be RepositoryRole" else empty end),
         (if .actor_id != 5 then "bypass actor_id must be 5 (repository admin)" else empty end),
         (if .bypass_mode != "pull_request" then "bypass_mode must be pull_request" else empty end)),
    ( ((.rules // []) | map(.type)) as $types
      | ( (if ($types | index("deletion")) == null then "missing rule: deletion" else empty end),
          (if ($types | index("non_fast_forward")) == null then "missing rule: non_fast_forward" else empty end),
          (if ($types | index("pull_request")) == null then "missing rule: pull_request" else empty end),
          (if ($types | length) != ($types | unique | length) then "duplicate rule types" else empty end),
          ($types[] | select(. != "deletion" and . != "non_fast_forward" and . != "pull_request")
             | "rule type outside this narrow policy: " + .) ) ),
    ((.rules // [])[] | select(.type == "pull_request") | .parameters
       | (if (.required_approving_review_count | type) != "number"
            or .required_approving_review_count < 0 or .required_approving_review_count > 10
            then "required_approving_review_count must be a number from 0 through 10" else empty end),
         (if (.dismiss_stale_reviews_on_push | type) != "boolean"
            then "dismiss_stale_reviews_on_push must be explicit" else empty end),
         (if (.required_review_thread_resolution | type) != "boolean"
            then "required_review_thread_resolution must be explicit" else empty end),
         (if (.require_code_owner_review | type) != "boolean"
            then "require_code_owner_review must be explicit" else empty end),
         (if (.require_last_push_approval | type) != "boolean"
            then "require_last_push_approval must be explicit" else empty end),
         (if (.require_extra_approval_for_unattributed_changes | type) != "boolean"
            then "require_extra_approval_for_unattributed_changes must be explicit (GitHub defaults it to true)" else empty end),
         (if ((.allowed_merge_methods // []) | length) == 0
            then "allowed_merge_methods must list at least one method" else empty end),
         ((.allowed_merge_methods // [])[]
            | select(. != "merge" and . != "squash" and . != "rebase")
            | "unsupported merge method: " + .)) ]
  | .[]
'

# ---------------------------------------------------------------- desired ----

build_desired() {
  local enforcement="$1" out="$WORK/desired.json"
  [ -f "$CONFIG" ] || die "config not found: $CONFIG"
  jq -e . "$CONFIG" >/dev/null 2>&1 || die "config is not valid JSON: $CONFIG"

  if [ -n "$enforcement" ]; then
    jq --arg e "$enforcement" '.enforcement = $e' "$CONFIG" > "$WORK/desired.raw.json"
  else
    cp "$CONFIG" "$WORK/desired.raw.json"
  fi

  local errors
  errors="$(jq -r --arg name "$NAME" --arg branch "$BRANCH" "$JQ_POLICY_ERRORS" \
    "$WORK/desired.raw.json")"
  if [ -n "$errors" ]; then
    printf 'config policy validation failed:\n' >&2
    printf '  - %s\n' "$errors" >&2
    exit 1
  fi

  jq -S "$JQ_NORMALIZE" "$WORK/desired.raw.json" > "$out"
}

# ---------------------------------------------------------------- current ----

# Sets RULESET_ID ("" when absent) and writes $WORK/current.json when present.
fetch_current() {
  local list
  list="$(api "repos/${REPO}/rulesets?includes_parents=true&targets=branch&per_page=100")"

  local count
  count="$(printf '%s' "$list" | jq --arg n "$NAME" '[.[] | select(.name == $n)] | length')"
  [ "$count" -le 1 ] || die "found $count rulesets named \"$NAME\"; refusing ambiguous reconciliation"

  printf '%s' "$list" | jq -r --arg n "$NAME" \
    '[.[] | select(.name != $n) | {id, name, enforcement, source_type, source}]' \
    > "$WORK/other-rulesets.json"

  if [ "$count" -eq 0 ]; then
    RULESET_ID=""
    return
  fi

  RULESET_ID="$(printf '%s' "$list" | jq -r --arg n "$NAME" \
    'first(.[] | select(.name == $n)) | .id')"

  local owner_type owner_source
  owner_type="$(printf '%s' "$list" | jq -r --arg n "$NAME" \
    'first(.[] | select(.name == $n)) | .source_type')"
  owner_source="$(printf '%s' "$list" | jq -r --arg n "$NAME" \
    'first(.[] | select(.name == $n)) | .source')"
  [ "$owner_type" = "Repository" ] ||
    die "ruleset \"$NAME\" is owned by $owner_type \"$owner_source\", not this repository"

  api "repos/${REPO}/rulesets/${RULESET_ID}?includes_parents=false" > "$WORK/current.raw.json"

  local non_neutral
  non_neutral="$(jq -r "$JQ_NON_NEUTRAL" "$WORK/current.raw.json")"
  [ "$non_neutral" = "false" ] ||
    die "the live ruleset has non-default reviewer settings; inspect it before overwriting"

  jq -e '.bypass_actors' "$WORK/current.raw.json" >/dev/null 2>&1 ||
    die "response omitted bypass_actors; re-authenticate with admin access"

  jq -S "$JQ_NORMALIZE" "$WORK/current.raw.json" > "$WORK/current.json"
}

# --------------------------------------------------------------- preflight ---

# Prints one blocker per line; empty output means preconditions passed.
preflight() {
  local repo_json blockers=""
  repo_json="$(api "repos/${REPO}")"

  local full_name visibility archived default_branch is_admin
  full_name="$(printf '%s' "$repo_json" | jq -r '.full_name')"
  visibility="$(printf '%s' "$repo_json" | jq -r '.visibility')"
  archived="$(printf '%s' "$repo_json" | jq -r '.archived')"
  default_branch="$(printf '%s' "$repo_json" | jq -r '.default_branch')"
  is_admin="$(printf '%s' "$repo_json" | jq -r '.permissions.admin // false')"

  [ "$full_name" = "$REPO" ] || blockers="${blockers}API resolved repository as \"$full_name\", not \"$REPO\"\n"
  [ "$visibility" = "public" ] || blockers="${blockers}repository is not public as this proposal assumes\n"
  [ "$archived" = "false" ] || blockers="${blockers}repository is archived\n"
  [ "$default_branch" = "$BRANCH" ] || blockers="${blockers}default branch is \"$default_branch\", expected \"$BRANCH\"\n"
  [ "$is_admin" = "true" ] || blockers="${blockers}authenticated identity lacks repository admin permission\n"

  # A ruleset may only allow a merge method the repository itself still enables.
  local methods method
  methods="$(jq -r "$JQ_MERGE_METHODS" "$WORK/desired.json")" ||
    die "could not read allowed_merge_methods from the desired payload"
  for method in $methods; do
    local key enabled
    case "$method" in
      merge)  key='.allow_merge_commit' ;;
      squash) key='.allow_squash_merge' ;;
      rebase) key='.allow_rebase_merge' ;;
      *)      continue ;;
    esac
    enabled="$(printf '%s' "$repo_json" | jq -r "$key // false")"
    [ "$enabled" = "true" ] ||
      blockers="${blockers}ruleset allows \"$method\" merges but the repository disables that method\n"
  done

  # Classic protection layers on top of rulesets and is easy to miss.
  if api "repos/${REPO}/branches/${BRANCH}/protection" >/dev/null 2>&1; then
    blockers="${blockers}classic branch protection already targets ${BRANCH}; reconcile it first\n"
  fi

  # Any active rule from a ruleset other than ours also layers.
  local foreign
  foreign="$(api "repos/${REPO}/rules/branches/${BRANCH}?per_page=100" \
    | jq -r --arg id "${RULESET_ID:-0}" \
        '[.[] | select((.ruleset_id | tostring) != $id)
              | "active rule \"\(.type)\" from ruleset \(.ruleset_id) already targets this branch"] | .[]')"
  [ -z "$foreign" ] || blockers="${blockers}${foreign}\n"

  [ -z "$blockers" ] || printf '%b' "$blockers"
}

# -------------------------------------------------------------------- plan ---

# Sets OPERATION and TOKEN.
compute_plan() {
  local desired_hash current_hash
  desired_hash="$(hash_file "$WORK/desired.json")"

  if [ -z "${RULESET_ID:-}" ]; then
    OPERATION="create"
    current_hash="absent"
  elif diff -q "$WORK/current.json" "$WORK/desired.json" >/dev/null 2>&1; then
    OPERATION="no-op"
    current_hash="$(hash_file "$WORK/current.json")"
  else
    OPERATION="update"
    current_hash="$(hash_file "$WORK/current.json")"
  fi

  TOKEN="$(printf '%s|%s|%s|%s|%s' "$REPO" "$NAME" "$OPERATION" "$desired_hash" "$current_hash" \
    | shasum -a 256 | cut -c1-16)"
}

show_diff() {
  if [ "$OPERATION" = "create" ]; then
    note "--- ruleset does not exist yet; full payload to be created ---"
    jq . "$WORK/desired.json" >&2
  elif [ "$OPERATION" = "no-op" ]; then
    note "--- live ruleset already matches the config ---"
  else
    note "--- diff: live (-) vs desired (+) ---"
    diff -u --label live "$WORK/current.json" --label desired "$WORK/desired.json" >&2 || true
  fi
}

require_confirm() {
  local supplied="$1"
  if [ -z "$supplied" ]; then
    note ""
    note "DRY RUN - nothing was changed."
    note "To proceed, re-run with: --confirm $TOKEN"
    exit 0
  fi
  [ "$supplied" = "$TOKEN" ] ||
    die "confirmation token does not match this fresh plan (expected $TOKEN); nothing was sent"
}

# ---------------------------------------------------------------- commands ---

cmd_validate() {
  build_desired "$ENFORCEMENT"
  printf 'config:  %s\n' "$CONFIG"
  printf 'sha256:  %s\n' "$(hash_file "$CONFIG")"
  printf 'valid:   yes (schema and policy checks passed, no network calls)\n'
}

cmd_plan() {
  build_desired "$ENFORCEMENT"
  fetch_current
  local blockers
  blockers="$(preflight)"
  compute_plan

  printf 'repository:   %s\n' "$REPO"
  printf 'branch:       %s\n' "$BRANCH"
  printf 'ruleset:      %s (%s)\n' "$NAME" "${RULESET_ID:-absent}"
  printf 'operation:    %s\n' "$OPERATION"
  printf 'api version:  %s\n' "$API_VERSION"

  local others
  others="$(jq -r 'if length == 0 then "" else (.[] | "  - \(.name) (id \(.id), \(.enforcement))") end' \
    "$WORK/other-rulesets.json")"
  if [ -n "$others" ]; then
    printf 'other branch rulesets (rules layer together):\n%s\n' "$others"
  fi

  show_diff

  if [ -n "$blockers" ]; then
    printf '\nblockers:\n' >&2
    printf '%s\n' "$blockers" | sed 's/^/  - /' >&2
    return 1
  fi
  return 0
}

cmd_apply() {
  cmd_plan || die "preconditions failed; refusing to apply"
  if [ "$OPERATION" = "no-op" ]; then
    note ""
    note "Nothing to do."
    exit 0
  fi
  require_confirm "$CONFIRM"

  if [ "$OPERATION" = "create" ]; then
    api --method POST "repos/${REPO}/rulesets" --input "$WORK/desired.raw.json" > "$WORK/result.json"
  else
    api --method PUT "repos/${REPO}/rulesets/${RULESET_ID}" --input "$WORK/desired.raw.json" > "$WORK/result.json"
  fi

  jq -S "$JQ_NORMALIZE" "$WORK/result.json" > "$WORK/result.norm.json"
  diff -q "$WORK/result.norm.json" "$WORK/desired.json" >/dev/null 2>&1 ||
    die "GitHub accepted the change but returned a different payload; inspect before doing anything else"

  printf '%s succeeded; ruleset id %s is now enforcement=%s\n' \
    "$OPERATION" "$(jq -r '.id' "$WORK/result.json")" "$(jq -r '.enforcement' "$WORK/result.json")"
}

cmd_disable() {
  build_desired ""
  fetch_current
  [ -n "${RULESET_ID:-}" ] || { note "Ruleset \"$NAME\" does not exist; nothing to disable."; exit 0; }

  local enforcement
  enforcement="$(jq -r '.enforcement' "$WORK/current.raw.json")"
  [ "$enforcement" != "disabled" ] || { note "Ruleset is already disabled."; exit 0; }

  OPERATION="disable"
  TOKEN="$(printf '%s|%s|disable|%s' "$REPO" "$NAME" "$RULESET_ID" | shasum -a 256 | cut -c1-16)"
  note "Will set enforcement=disabled on ruleset $RULESET_ID (payload otherwise untouched)."
  require_confirm "$CONFIRM"

  api --method PUT "repos/${REPO}/rulesets/${RULESET_ID}" \
    -f enforcement=disabled > "$WORK/result.json"
  [ "$(jq -r '.enforcement' "$WORK/result.json")" = "disabled" ] ||
    die "GitHub did not report the ruleset as disabled; inspect it now"
  printf 'ruleset %s is now disabled; the rules and their history are preserved\n' "$RULESET_ID"
}

cmd_delete() {
  build_desired ""
  fetch_current
  [ -n "${RULESET_ID:-}" ] || { note "Ruleset \"$NAME\" does not exist; nothing to delete."; exit 0; }

  OPERATION="delete"
  TOKEN="$(printf '%s|%s|delete|%s' "$REPO" "$NAME" "$RULESET_ID" | shasum -a 256 | cut -c1-16)"
  note "Deletion is irreversible. Prefer 'disable' unless you are restoring a"
  note "verified pre-creation state. Current ruleset $RULESET_ID:"
  jq -S "$JQ_NORMALIZE" "$WORK/current.raw.json" >&2
  require_confirm "$CONFIRM"

  api --method DELETE "repos/${REPO}/rulesets/${RULESET_ID}" >/dev/null
  printf 'ruleset %s deleted\n' "$RULESET_ID"
}

# -------------------------------------------------------------------- main ---

COMMAND="plan"
ENFORCEMENT=""
CONFIRM=""

main() {
  if [ "$#" -gt 0 ] && case "$1" in -*) false ;; *) true ;; esac; then
    COMMAND="$1"
    shift
  fi

  while [ "$#" -gt 0 ]; do
    case "$1" in
      --enforcement) [ "$#" -ge 2 ] || die "--enforcement needs a value"; ENFORCEMENT="$2"; shift 2 ;;
      --confirm)     [ "$#" -ge 2 ] || die "--confirm needs a token"; CONFIRM="$2"; shift 2 ;;
      -h|--help)     usage; exit 0 ;;
      *)             usage; die "unexpected argument: $1" ;;
    esac
  done

  case "$ENFORCEMENT" in
    ""|active|disabled) ;;
    *) die "--enforcement must be active or disabled, got \"$ENFORCEMENT\"" ;;
  esac

  command -v gh >/dev/null 2>&1 || die "gh not found; install GitHub CLI and run: gh auth login"
  command -v jq >/dev/null 2>&1 || die "jq not found"

  case "$COMMAND" in
    validate) cmd_validate ;;
    plan)     cmd_plan ;;
    apply)    cmd_apply ;;
    disable)  cmd_disable ;;
    delete)   cmd_delete ;;
    -h|--help|help) usage ;;
    *)        usage; die "unknown command: $COMMAND" ;;
  esac
}

# Only dispatch when executed. Sourcing exposes the functions and jq programs
# to the test suite without running a command.
if [ "${BASH_SOURCE[0]}" = "$0" ]; then
  main "$@"
fi
