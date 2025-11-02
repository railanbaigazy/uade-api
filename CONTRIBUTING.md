# Contributing to Uade API

This document outlines how our team writes commits, pull requests, and tests code.

## Development Workflow

### 1. Create a new branch: (type and name are variable)

```bash
git checkout -b <type>/<name>
```

Examples of branch names:

- `feat/user-auth`
- `fix/transactions-rollback`
- `chore/update-dependencies`

### 2. Make changes and write tests

### 3. Run all tests before pushing:

```bash
make test
```

### 4. Commit using the correct format (see below)

### 5. Push your branch and open a Pull Request (PR)

## Commit Message Convention

We follow the Conventional Commits specification.

### Format

`type(optional-scope): short description`

### Examples

- `feat(users): add user registration endpoint`
- `fix(db): fix foreign key reference in agreements table`
- `refactor(handlers): simplify JSON responses`
- `chore(ci): add github actions workflow`
- `docs: update setup instructions`
- `test: add integration test for healthz route`

### Commit Types

| Type     |              Meaning               |
| :------- | :--------------------------------: |
| feat     |            New feature             |
| fix      |              Bug fix               |
| refactor | Code refactor (no behavior change) |
| chore    |  Maintenance (Docker, deps, etc.)  |
| test     |        Test-related changes        |
| docs     |         Documentation only         |

## Pull Request Naming

Use the same convention as in commit naming for PRs:

Example:

`feat(agreements): implement agreement creation endpoint`

## Review Process

1. Open a PR -> it must pass CI tests.
2. Another team member reviews your code.
3. If approved, the PR is merged into main.
