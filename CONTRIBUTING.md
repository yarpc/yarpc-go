# Contributing

We welcome contributions from the community. Here are some guidelines to make it easier for everyone.

## Default branch renaming

For a long time, we had two protected branches: master and dev. Latter was used as default one.
We also used git-flow to manage our releases.

One Feb 10, 2025 `master` was renamed to `archived-releases`, and `dev` was renamed to `main`.

Moreover, we stopped using git-flow and switched to a trunk-based development model. In realm of this repo,
we use feature branches for development, pull requests to merge changes to main, and release branches for
cutting new releases. I.e. no tags are created on main branch.

If your local copy still has a `dev` branch, please use following commands to rename it to `main`:

```
git checkout dev
git branch -m dev main
git fetch origin
git branch -u origin/main main
git remote set-head origin -a
```

## Creating a Pull Request

Please use `RELEASE NOTES:` in a PR summary to write all significant changes that you've made.
This section will be used later to compile a release notes.

Please use `N/a` if there are no release notes.

## Development

### Setup

To start developing with yarpc-go, make a fork via the Github UI, and clone it locally:

```
git clone https://github.com/{github-username}/yarpc-go.git go.uber.org/yarpc
```

### Running Tests

To run tests into a pre-configured docker container, run the following command:
```
make test
```

To run tests locally, run the following command:
```
SUPPRESS_DOCKER=1 make test
```

Happy development!