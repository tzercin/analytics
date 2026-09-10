#!/usr/bin/env bash
#
# Offline tests for scripts/github-ruleset.sh.
#
# These exercise the config validation and argument handling only. Every case
# runs the "validate" command or a dry run, so no test here contacts GitHub.

set -uo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
SCRIPT="$HERE/github-ruleset.sh"
REAL_CONFIG="$HERE/../.github/rulesets/protect-master.json"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

PASS=0
FAIL=0

ok() { PASS=$((PASS + 1)); printf 'ok   %s\n' "$1"; }
no() { FAIL=$((FAIL + 1)); printf 'FAIL %s\n     %s\n' "$1" "$2"; }

# run_validate <config-file> -> captures $OUT and $RC
run_validate() {
  OUT="$(RULESET_CONFIG="$1" "$SCRIPT" validate 2>&1)"
  RC=$?
}

# expect_valid <name> <config>
expect_valid() {
  run_validate "$2"
  if [ "$RC" -eq 0 ]; then ok "$1"; else no "$1" "expected exit 0, got $RC: $OUT"; fi
}

# expect_invalid <name> <config> <substring>
expect_invalid() {
  run_validate "$2"
  if [ "$RC" -eq 0 ]; then
    no "$1" "expected non-zero exit, got 0"
  elif printf '%s' "$OUT" | grep -qF "$3"; then
    ok "$1"
  else
    no "$1" "expected message containing '$3', got: $OUT"
  fi
}

# mutate <jq-filter> -> path to a modified copy of the real config
mutate() {
  local out
  out="$WORK/cfg-$RANDOM.json"
  jq "$1" "$REAL_CONFIG" > "$out"
  printf '%s' "$out"
}

printf '== committed config ==\n'
expect_valid "the committed config passes validation" "$REAL_CONFIG"

printf '\n== structural policy ==\n'
expect_invalid "rejects a renamed ruleset" \
  "$(mutate '.name = "something-else"')" 'name must be'
expect_invalid "rejects a tag target" \
  "$(mutate '.target = "tag"')" 'target must be'
expect_invalid "rejects Enterprise-only evaluate enforcement" \
  "$(mutate '.enforcement = "evaluate"')" 'Enterprise-only'
expect_invalid "rejects a wildcard branch target" \
  "$(mutate '.conditions.ref_name.include = ["refs/heads/**"]')" 'must be exactly'
expect_invalid "rejects extra targeted branches" \
  "$(mutate '.conditions.ref_name.include += ["refs/heads/release"]')" 'must be exactly'
expect_invalid "rejects a non-empty exclude list" \
  "$(mutate '.conditions.ref_name.exclude = ["refs/heads/tmp"]')" 'exclude must be empty'

printf '\n== bypass safety ==\n'
expect_invalid "rejects an unrestricted always bypass" \
  "$(mutate '.bypass_actors[0].bypass_mode = "always"')" 'bypass_mode must be pull_request'
expect_invalid "rejects a non-admin bypass role id" \
  "$(mutate '.bypass_actors[0].actor_id = 2')" 'actor_id must be 5'
expect_invalid "rejects an extra bypass actor" \
  "$(mutate '.bypass_actors += [{"actor_id":1,"actor_type":"RepositoryRole","bypass_mode":"pull_request"}]')" \
  'exactly the admin recovery entry'
expect_invalid "rejects an unsupported actor type" \
  "$(mutate '.bypass_actors[0].actor_type = "Integration"')" 'actor_type must be RepositoryRole'

printf '\n== required rules ==\n'
expect_invalid "rejects dropping the deletion rule" \
  "$(mutate '.rules |= map(select(.type != "deletion"))')" 'missing rule: deletion'
expect_invalid "rejects dropping the force-push rule" \
  "$(mutate '.rules |= map(select(.type != "non_fast_forward"))')" 'missing rule: non_fast_forward'
expect_invalid "rejects dropping the pull request rule" \
  "$(mutate '.rules |= map(select(.type != "pull_request"))')" 'missing rule: pull_request'
expect_invalid "rejects an unreviewed extra rule type" \
  "$(mutate '.rules += [{"type":"required_signatures"}]')" 'outside this narrow policy'
expect_invalid "rejects duplicate rule types" \
  "$(mutate '.rules += [{"type":"deletion"}]')" 'duplicate rule types'

printf '\n== pull request parameters ==\n'
expect_invalid "rejects an out-of-range approval count" \
  "$(mutate '(.rules[] | select(.type=="pull_request") | .parameters.required_approving_review_count) = 99')" \
  'from 0 through 10'
expect_invalid "rejects an implicit conversation-resolution setting" \
  "$(mutate '(.rules[] | select(.type=="pull_request") | .parameters) |= del(.required_review_thread_resolution)')" \
  'required_review_thread_resolution must be explicit'
expect_invalid "rejects an implicit stale-review setting" \
  "$(mutate '(.rules[] | select(.type=="pull_request") | .parameters) |= del(.dismiss_stale_reviews_on_push)')" \
  'dismiss_stale_reviews_on_push must be explicit'
expect_invalid "rejects an implicit unattributed-changes setting" \
  "$(mutate '(.rules[] | select(.type=="pull_request") | .parameters) |= del(.require_extra_approval_for_unattributed_changes)')" \
  'require_extra_approval_for_unattributed_changes must be explicit'
expect_invalid "rejects an empty merge-method list" \
  "$(mutate '(.rules[] | select(.type=="pull_request") | .parameters.allowed_merge_methods) = []')" \
  'at least one method'
expect_invalid "rejects an unsupported merge method" \
  "$(mutate '(.rules[] | select(.type=="pull_request") | .parameters.allowed_merge_methods) = ["fast-forward"]')" \
  'unsupported merge method'

printf '\n== zero approvals is allowed (documented reviewer-gate option 3) ==\n'
expect_valid "accepts zero required approvals" \
  "$(mutate '(.rules[] | select(.type=="pull_request") | .parameters.required_approving_review_count) = 0')"

printf '\n== enforcement override ==\n'
OUT="$(RULESET_CONFIG="$REAL_CONFIG" "$SCRIPT" validate --enforcement disabled 2>&1)"; RC=$?
if [ "$RC" -eq 0 ]; then ok "accepts --enforcement disabled"; else no "accepts --enforcement disabled" "$OUT"; fi

OUT="$(RULESET_CONFIG="$REAL_CONFIG" "$SCRIPT" validate --enforcement evaluate 2>&1)"; RC=$?
if [ "$RC" -ne 0 ] && printf '%s' "$OUT" | grep -qF 'must be active or disabled'; then
  ok "rejects --enforcement evaluate"
else
  no "rejects --enforcement evaluate" "$OUT"
fi

printf '\n== malformed input and CLI handling ==\n'
printf '{"name": ' > "$WORK/truncated.json"
expect_invalid "rejects malformed JSON" "$WORK/truncated.json" 'not valid JSON'
expect_invalid "rejects a missing config file" "$WORK/absent.json" 'config not found'

OUT="$("$SCRIPT" frobnicate 2>&1)"; RC=$?
if [ "$RC" -ne 0 ] && printf '%s' "$OUT" | grep -qF 'unknown command'; then
  ok "rejects an unknown command"
else
  no "rejects an unknown command" "$OUT"
fi

OUT="$("$SCRIPT" validate --confirm 2>&1)"; RC=$?
if [ "$RC" -ne 0 ] && printf '%s' "$OUT" | grep -qF 'needs a token'; then
  ok "rejects --confirm without a value"
else
  no "rejects --confirm without a value" "$OUT"
fi

printf '\n== validate performs no network calls ==\n'
# Shadow gh with a failing stub: validate must still succeed.
mkdir -p "$WORK/bin"
cat > "$WORK/bin/gh" <<'STUB'
#!/bin/sh
echo "gh was called during an offline command" >&2
exit 97
STUB
chmod +x "$WORK/bin/gh"
OUT="$(PATH="$WORK/bin:$PATH" RULESET_CONFIG="$REAL_CONFIG" "$SCRIPT" validate 2>&1)"; RC=$?
if [ "$RC" -eq 0 ]; then ok "validate never invokes gh"; else no "validate never invokes gh" "$OUT"; fi

printf '\n== internal jq programs (sourced, no dispatch) ==\n'
# shellcheck source-path=SCRIPTDIR source=github-ruleset.sh
. "$SCRIPT"

# Regression: this filter once read .allowed_merge_methods off the rule instead
# of off .parameters, so it errored to empty and the preflight check silently
# validated nothing.
GOT="$(jq -r "$JQ_MERGE_METHODS" "$REAL_CONFIG" | sort | tr '\n' ' ')"
if [ "$GOT" = "merge rebase squash " ]; then
  ok "merge-method filter reads the parameters path"
else
  no "merge-method filter reads the parameters path" "got '$GOT'"
fi

# Normalization must absorb GitHub's echoed neutral defaults so that an
# unchanged ruleset diffs as a no-op rather than perpetual drift.
jq '(.rules[] | select(.type=="pull_request") | .parameters) +=
      {dismissal_restriction: {allowed_actors: [], enabled: false},
       ignore_approvals_from_contributors: false,
       required_reviewers: []}
    | . + {id: 42, source_type: "Repository", source: "tzercin/analytics"}' \
  "$REAL_CONFIG" > "$WORK/remote-neutral.json"

if diff -q <(jq -S "$JQ_NORMALIZE" "$WORK/remote-neutral.json") \
           <(jq -S "$JQ_NORMALIZE" "$REAL_CONFIG") >/dev/null 2>&1; then
  ok "normalization absorbs neutral server defaults"
else
  no "normalization absorbs neutral server defaults" "$(diff <(jq -S "$JQ_NORMALIZE" "$WORK/remote-neutral.json") <(jq -S "$JQ_NORMALIZE" "$REAL_CONFIG"))"
fi

# Rule and merge-method ordering differences are not policy differences.
jq '.rules |= reverse
    | (.rules[] | select(.type=="pull_request") | .parameters.allowed_merge_methods) |= reverse' \
  "$REAL_CONFIG" > "$WORK/reordered.json"
if diff -q <(jq -S "$JQ_NORMALIZE" "$WORK/reordered.json") \
           <(jq -S "$JQ_NORMALIZE" "$REAL_CONFIG") >/dev/null 2>&1; then
  ok "normalization is order-insensitive"
else
  no "normalization is order-insensitive" "reordering changed the normal form"
fi

# Regression: GitHub enables require_extra_approval_for_unattributed_changes by
# default, so it is declared policy and a difference in it must show as a diff
# rather than being stripped. It was previously deleted during normalization,
# which hid the fact that the live ruleset disagreed with the config.
jq '(.rules[] | select(.type=="pull_request")
      | .parameters.require_extra_approval_for_unattributed_changes) = false' \
  "$REAL_CONFIG" > "$WORK/remote-extra-off.json"
if diff -q <(jq -S "$JQ_NORMALIZE" "$WORK/remote-extra-off.json") \
           <(jq -S "$JQ_NORMALIZE" "$REAL_CONFIG") >/dev/null 2>&1; then
  no "unattributed-changes difference surfaces as a diff" "normalization hid a real policy difference"
else
  ok "unattributed-changes difference surfaces as a diff"
fi

# A remote that omits the field means GitHub's default (true), not false.
jq '(.rules[] | select(.type=="pull_request") | .parameters)
      |= del(.require_extra_approval_for_unattributed_changes)' \
  "$REAL_CONFIG" > "$WORK/remote-extra-absent.json"
if diff -q <(jq -S "$JQ_NORMALIZE" "$WORK/remote-extra-absent.json") \
           <(jq -S "$JQ_NORMALIZE" "$REAL_CONFIG") >/dev/null 2>&1; then
  ok "an omitted unattributed-changes field defaults to true"
else
  no "an omitted unattributed-changes field defaults to true" "expected the default to match the declared true"
fi

# A non-neutral remote setting must NOT be silently normalized away.
jq '(.rules[] | select(.type=="pull_request") | .parameters)
      += {ignore_approvals_from_contributors: true}' "$REAL_CONFIG" > "$WORK/remote-hostile.json"
if [ "$(jq -r "$JQ_NON_NEUTRAL" "$WORK/remote-hostile.json")" = "true" ]; then
  ok "non-neutral remote settings are detected"
else
  no "non-neutral remote settings are detected" "expected true"
fi
if [ "$(jq -r "$JQ_NON_NEUTRAL" "$WORK/remote-neutral.json")" = "false" ]; then
  ok "neutral remote settings are not flagged"
else
  no "neutral remote settings are not flagged" "expected false"
fi

printf '\n%s passed, %s failed\n' "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]
