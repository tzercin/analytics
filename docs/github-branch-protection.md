# Default-branch ruleset proposal

> [!IMPORTANT]
> This is a local proposal. It has **not** changed GitHub, and it must not be
> applied, committed, pushed, or published without an explicit maintainer
> review and approval for that separate action.

The intended end state is the declarative repository ruleset in
[`../.github/rulesets/protect-master.json`](../.github/rulesets/protect-master.json).
That file is the policy. [`../scripts/github-ruleset.sh`](../scripts/github-ruleset.sh)
is a thin `gh` wrapper that validates the deliberately narrow schema and shows a
read-only diff before it can send a mutation.

The wrapper exists only for the parts that are genuinely awkward as one-liners:
normalizing GitHub's echoed default fields so an unchanged ruleset diffs as a
no-op, and refusing to apply when protection is layered. Everything it does is a
`gh api` call you can run by hand; the equivalent raw commands are shown below.

## Verified starting state

The repository and GitHub settings were re-inspected read-only through
`2026-09-10T10:03:13Z` using GitHub REST API version `2026-03-10`, GitHub CLI,
and local Git:

- `tzercin/analytics` resolved as a public, unarchived personal-account
  repository whose default branch is `master`.
- Local `HEAD`, `origin/master`, and the live `refs/heads/master` all resolved to
  `035f87515d7dc79865f68c2cd4f4702b6e1e37d4`.
- The repository ruleset list was empty, the effective-rules endpoint for
  `master` returned `[]`, `gh ruleset check master` reported "0 rules apply to
  branch master", and the classic branch-protection endpoint returned GitHub's
  `404 Branch not protected` response.
- There were no open pull requests and no GitHub Actions workflows
  (`total_count: 0`). The `master` head commit had zero check runs and a
  `pending` combined status with zero status contexts. No stable check name
  therefore exists to require.
- The collaborator inspection returned only the repository owner with the
  `admin` role. A second write-permission reviewer is not currently evident.
- The API version was confirmed supported: GitHub rejects an unknown version and
  names `2026-03-10` as the most recent supported version.

These are time-sensitive source facts, not enduring assumptions. Re-run the
inspection before every rollout attempt. The principal read-only calls were:

```sh
git ls-remote --heads origin
gh api -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' repos/tzercin/analytics
gh api -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  'repos/tzercin/analytics/rulesets?includes_parents=true&targets=branch&per_page=100'
gh api -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  'repos/tzercin/analytics/rules/branches/master?per_page=100'
gh api -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  repos/tzercin/analytics/branches/master/protection
gh api -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  'repos/tzercin/analytics/pulls?state=open&per_page=100'
gh api -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  repos/tzercin/analytics/actions/workflows
gh api -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  repos/tzercin/analytics/commits/master/check-runs
gh api -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  repos/tzercin/analytics/commits/master/status
gh api -H 'Accept: application/vnd.github+json' \
  -H 'X-GitHub-Api-Version: 2026-03-10' \
  'repos/tzercin/analytics/collaborators?per_page=100'
```

No existing or open superseding branch-protection work was found.

## Mechanism decision

**GitHub source facts** (each verified against GitHub documentation on
2026-09-10):

- Availability: "Rulesets are available in public repositories with GitHub Free
  and GitHub Free for organizations, and in public and private repositories with
  GitHub Pro, GitHub Team, and GitHub Enterprise Cloud." This repository is
  public, so repository rulesets are available on its plan. Classic protected
  branches are likewise available for public repositories.
- Layering: "if multiple rulesets target the same branch or tag in a repository,
  the rules in each of these rulesets are aggregated," and where the same rule
  appears in more than one form, "the most restrictive version of the rule
  applies."
- Classic contrast: "Only a single branch protection rule can apply at a time,
  which means it can be difficult to know which rule will apply when multiple
  versions of a rule target the same branch." GitHub notes this restriction
  "does not apply to rulesets."
- Enforcement modes: the REST API accepts `disabled`, `active`, and `evaluate`,
  but documents that "`evaluate` is only available with GitHub Enterprise."
  `evaluate` is therefore *not* a rollout option here; staging must use
  `disabled`.
- The REST API supports list, create, get, update, delete, effective-rule, and
  history operations, so a ruleset can be inspected and diffed declaratively.

**Recommendation:** use one repository-level branch ruleset rather than adding
a classic rule. It is easier to inspect through the current API, can be staged
as disabled, has history, and fails less opaquely if future rules are layered.
The tool blocks apply if it finds classic protection or active rules from any
other ruleset on `master`; it reports other branch rulesets for manual review.

Authoritative references:

- [About rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets)
- [Available rules for rulesets](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/available-rules-for-rulesets)
- [REST API endpoints for repository rules](https://docs.github.com/en/rest/repos/rules?apiVersion=2026-03-10)
- [GitHub REST API versions](https://docs.github.com/en/rest/about-the-rest-api/api-versions)

## Proposed policy

| Setting | Proposed value | Rationale |
| --- | --- | --- |
| Target | Exact `refs/heads/master` | Protect only the verified default branch; avoid a surprising wildcard. |
| Final enforcement | `active` | The committed file represents the intended end state. Rollout first overrides this to `disabled`. |
| Pull request | Required | Prevent routine direct updates to the default branch. |
| Approvals | `1` | Conservative independent-review baseline. See the reviewer gate below. |
| Stale approvals | Dismiss on reviewable push | A changed diff must be reviewed again. |
| Conversations | Resolution required | Unresolved review threads block merge. |
| Last-push approval | Not additionally required | Avoid a second independent-review constraint in a currently single-maintainer repository; stale-review dismissal still covers changed diffs. |
| Code-owner review | Not required | No `CODEOWNERS` file or ownership process exists. |
| Unattributed-change approval | `true` | GitHub enables this by default on new and existing rulesets. It adds one approval for a Copilot PR not attributed to a person. Declared explicitly so the file matches the live state rather than drifting silently. |
| Merge methods | Merge, squash, and rebase | Matches all three methods currently enabled at repository level; this proposal does not change merge settings. |
| Force pushes | Blocked by `non_fast_forward` | Protect history. |
| Deletion | Blocked by `deletion` | Protect the default ref. |
| Status checks | None | No workflow, check run, or status context currently exists. Requiring an invented name would lock every PR. |
| Bypass | Repository administrators, pull-request mode only | Provides an auditable PR recovery path without granting an unrestricted `always` or unaudited `exempt` bypass. |

`RepositoryRole` actor ID `5` is the base `admin` role. The REST schema requires
an ID but does not publish the base-role mapping inline; GitHub's maintained
[Terraform provider documentation](https://github.com/integrations/terraform-provider-github/blob/main/docs/resources/repository_ruleset.md#bypass-actors)
records `admin -> 5`. Confirm that the GitHub import/settings preview renders
this entry as **Repository admin** before activating it. Do not proceed if it
does not.

### Reviewer gate

The one-approval recommendation is intentionally stricter than current practice.
The current collaborator response does not show a second write-permission
reviewer, and GitHub counts required approvals from reviewers with write
permission. Before activation, the maintainer must choose one of these explicitly:

1. add and use a trusted reviewer with the least suitable write permission; or
2. accept that solo changes require the administrator's explicit PR bypass; or
3. review and change the proposal to zero required approvals while retaining the
   PR and conversation-resolution requirements.

The PR-only administrator bypass prevents lockout of PR merging, but it should be
a recovery mechanism rather than an unexamined routine. If a repair cannot be
performed through a PR, an administrator can disable the ruleset through the
API or repository settings.

## Tool behavior and prerequisites

Prerequisites:

- GitHub CLI (`gh`) installed and authenticated (`gh auth login`);
- `jq` and `shasum`;
- metadata read permission for inspection; and
- repository **Administration: write** permission for create/update/delete.

`gh` owns authentication. The script never requests, prints, or places a token in
process arguments. Keep tokens in the GitHub CLI credential store or an approved
secret store; never add them to this repository.

The default subcommand is read-only `plan`. The script pins the REST API version
and fails closed on duplicate ruleset names, a missing `bypass_actors` field, an
unexpected target or default branch, a merge method the repository disables,
non-neutral server-added reviewer settings, classic protection, or another active
ruleset targeting `master`.

GitHub adds neutral fields to ruleset responses even when they were omitted from
the request. Normalization removes only recognized neutral defaults (for example,
an empty disabled review-dismissal restriction) so that an unchanged ruleset
diffs as a no-op. A non-neutral value stops reconciliation rather than silently
overwriting it.

Behavior is covered by [`../scripts/github-ruleset_test.sh`](../scripts/github-ruleset_test.sh),
an offline suite that shadows `gh` with a failing stub to prove the offline paths
never reach the network.

## Validate, inspect, and dry-run

All commands below are read-only unless they include a `--confirm` token that
matches the immediately preceding dry run.

```sh
# Offline schema and policy validation; makes no API calls.
scripts/github-ruleset.sh validate

# Live read-only inspection and create/update/no-op diff.
scripts/github-ruleset.sh plan

# Offline test suite.
scripts/github-ruleset_test.sh
```

`plan` exits non-zero and lists blockers if any precondition fails. Passing
preconditions is not maintainer approval.

The same inspection with raw `gh`, if you would rather not use the wrapper:

```sh
gh api -H 'X-GitHub-Api-Version: 2026-03-10' \
  'repos/tzercin/analytics/rulesets?includes_parents=true&targets=branch&per_page=100'
gh api -H 'X-GitHub-Api-Version: 2026-03-10' \
  'repos/tzercin/analytics/rules/branches/master?per_page=100'
gh ruleset check master --repo tzercin/analytics
```

## Staged rollout (requires separate approval)

Do not run any `--confirm` command until the reviewer gate, administrator bypass
preview, diff, and recovery path have been reviewed.

### 1. Create or reconcile it as disabled

The source file remains the final active policy; `--enforcement disabled` makes
the staged payload explicit without hand-editing a derived copy.

```sh
# Dry run: prints the diff and a confirmation token. Changes nothing.
scripts/github-ruleset.sh apply --enforcement disabled

# MUTATION: only after explicit approval for this action.
scripts/github-ruleset.sh apply --enforcement disabled --confirm <TOKEN>

# Read-only verification.
scripts/github-ruleset.sh plan --enforcement disabled   # expect operation: no-op
```

With no `--confirm`, even `apply` is a dry run. With a wrong or stale token it
fails without sending anything. The token is derived from the repository, ruleset
name, operation, and the hashes of the desired and current payloads, and the
command recomputes the live plan before comparing, so a token stops matching as
soon as either side changes.

The raw equivalent, if you prefer to do it by hand:

```sh
jq '.enforcement = "disabled"' .github/rulesets/protect-master.json > /tmp/staged.json
gh api -H 'X-GitHub-Api-Version: 2026-03-10' \
  --method POST repos/tzercin/analytics/rulesets --input /tmp/staged.json
```

Inspect the disabled result in GitHub's Rulesets settings and verify the exact
target, the **Repository admin / For pull requests only** bypass, and all policy
values. Disabled is not GitHub's Enterprise-only evaluation mode and produces no
enforcement signal.

### 2. Activate

```sh
scripts/github-ruleset.sh apply            # dry run; prints the token

# MUTATION: only after a new explicit approval for activation.
scripts/github-ruleset.sh apply --confirm <TOKEN>

# Read-only verification from both the declarative diff and effective rules.
scripts/github-ruleset.sh plan             # expect operation: no-op
gh ruleset check master --repo tzercin/analytics
```

Then open a small test PR from a non-default branch. Confirm that direct update,
force-push, deletion, approval, stale-review, and conversation behavior matches
the reviewed policy. Do not experiment destructively on `master`.

### 3. Add checks later, not speculatively

First add a separately reviewed CI workflow. Observe its actual, stable check
context on pull requests and the default branch, ensure job names are unique,
then extend the schema, config, tests, and documentation in another change. Do
not type a hoped-for check name into this ruleset.

## Rollback

Disabling is the preferred first response because it preserves the ruleset and
history. Running `disable` without confirmation is a dry-run that prints a token;
the confirmed call updates only `enforcement`, not the rest of the payload.

```sh
scripts/github-ruleset.sh disable          # dry run; prints the token

# MUTATION: only after explicit rollback approval.
scripts/github-ruleset.sh disable --confirm <TOKEN>
scripts/github-ruleset.sh plan --enforcement disabled   # expect operation: no-op
```

`disable` sends only `{"enforcement":"disabled"}`, leaving the rest of the
payload and the ruleset history intact.

If and only if this rollout created the ruleset from the verified absent state,
deletion restores that structural pre-state. Export it first; prefer disabling.

```sh
id=$(gh api -H 'X-GitHub-Api-Version: 2026-03-10' \
  'repos/tzercin/analytics/rulesets?targets=branch&per_page=100' \
  --jq 'first(.[] | select(.name == "protect-master")) | .id')
gh api -H 'X-GitHub-Api-Version: 2026-03-10' \
  "repos/tzercin/analytics/rulesets/$id" > /tmp/protect-master.before-delete.json

scripts/github-ruleset.sh delete           # dry run; prints the token

# IRREVERSIBLE MUTATION: requires separate explicit approval.
scripts/github-ruleset.sh delete --confirm <TOKEN>
scripts/github-ruleset.sh plan             # should again propose create
```

Never commit the temporary exports. If this proposal updates a pre-existing
ruleset in the future, use the GitHub ruleset-history API and the reviewed prior
version rather than assuming deletion is a valid rollback.

## Limitations and unresolved decisions

- **Approval workflow:** a routine independent reviewer is not presently evident;
  resolve the reviewer gate before activation.
- **Administrator role ID:** verify the rendered bypass actor in GitHub's disabled
  ruleset/settings preview before activation.
- **No server-side validation-only call:** `plan` is entirely GET-based. Creating
  a disabled ruleset is still a real GitHub mutation.
- **No Enterprise evaluation mode:** the public personal repository can use
  `active` or `disabled`, not `evaluate`.
- **API race window:** the confirmation is recomputed from fresh state inside the
  apply invocation, but GitHub's update endpoint does not expose an atomic
  compare-and-swap for the full payload. Re-inspect after every mutation.
- **Layering:** a future organization/enterprise rule or classic protection can
  make the effective policy stricter. The tool reports and blocks known active
  layering rather than trying to weaken it.
- **Status checks:** protection is incomplete against untested changes until a
  real CI workflow is established and its observed context is separately added.

