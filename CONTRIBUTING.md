# Contributing

This document describes the rules every change must follow to get merged.

## Branch protection

The `master` branch is protected:

- Direct pushes are rejected. Every change goes through a Pull Request.
- A PR needs at least one approval from the code owners
  (see `.github/CODEOWNERS`).
- All required CI checks must be green.
- A new push to the PR dismisses previous approvals, so ask for a re-review
  after updating the branch.
- All review conversations must be resolved before merging.
- Every commit must have a verified signature (see below).

## Commit signing

GitHub blocks merging a PR that contains unsigned commits. The simplest setup
is SSH signing with the key you already use for GitHub:

```sh
git config --global gpg.format ssh
git config --global user.signingkey ~/.ssh/id_ed25519.pub
git config --global commit.gpgsign true
```

Then add the same public key on GitHub a second time as a signing key:
Settings -> SSH and GPG keys -> New SSH key -> Key type: Signing Key. This is
a separate entry from your authentication key, even though the key material
is the same.

If a PR is blocked because already pushed commits are unsigned, re-sign them
and force-push:

```sh
git rebase -f origin/master
git push --force-with-lease
```

## Workflow

1. Create a feature branch from `master`.
2. Make the change and keep the local checks green (see below).
3. Open a Pull Request against `master` and describe what changed and why.
4. Wait for green CI, get an approval from a code owner, resolve all
   conversations, merge.

## Local checks

```sh
make build
make linters
go test -race ./...
```

`make build` installs the provider into the local plugin directory for
`dev_overrides`. Integration tests run with
`go test -tags=integration ./edgecenter/integrationtest/...` and need no
credentials. Acceptance tests (`make test_cloud_resource`,
`make test_not_cloud`, ...) create real cloud resources and require a
Vault-sourced `.env`; they run in CI, do not run them locally without a
reason. When the provider schema changes, regenerate the docs with
`make docs` and commit the result.
