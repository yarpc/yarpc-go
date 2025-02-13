# Release process
===============

> **NOTE**: Don't do any of this before validating with a test service. Check
> the internal release doc (ask someone about it) for more information.

This document outlines how to create a release of yarpc-go.

## Process description

We're using trunk-based development model. It means, all the development is done in feature branches,
and changes are merged to `main` branch via pull requests.

When it's time to cut a new release, we create a release branch from `main`, update `version.go`,
and tag the release.

We don't use tags on `main` branch, and we never merge release branches back to `main`.

## Prerequisites
-------------

Make sure you have `gh` installed.

```
brew install gh
```

Authorize `gh` with GitHub.

```
gh auth login
? What account do you want to log into? GitHub.com
? What is your preferred protocol for Git operations on this host? SSH
? Upload your SSH public key to your GitHub account? Skip
? How would you like to authenticate GitHub CLI? Login with a web browser
```

## Releasing

- Decide what's the next version number. It should be a semver-compatible string.
Usually, we increase minor version number for new features, and patch version number for bug fixes.

I.e. 1.2.0 -> 1.3.0 for new features, and 1.2.0 -> 1.2.1 for bug fixes.

You may get last release version by running:

  ```
    gh release list --exclude-drafts --exclude-pre-releases --json "tagName" | jq -r '.[0].tagName'
  ```

- Set environment variable with the version you want to release.

  ```
  VERSION=1.0.0
  ```

- Compile release notes.

  ```
  ./etc/bin/release/cmd-format-release-notes.sh \
        $(git log --pretty=format:"%H" \
              $(git merge-base main \
                  $(gh release list --exclude-drafts --exclude-pre-releases --json "isLatest,tagName" | \
                       jq -r '.[] | select( .isLatest ) | .tagName') \
               )..HEAD) | tee /tmp/yarpc-release-notes.txt
  ```

- Please format release notes:

    ```
      nano /tmp/yarpc-release-notes.txt
      
      # for vim users
      vim /tmp/yarpc-release-notes.txt
    ```


- Create a release branch from the specified branch.

    ```
    git checkout -b release/v$VERSION
    ```

- Update version.go with the new version.

    ```
    ./etc/bin/release/cmd-update-version.sh $VERSION
    ```

- Commit this change and push the branch to GitHub.

    ```
    git add version.go
    git commit -s -m "Prepare release v$VERSION"
    ```

- Cut a new release, using the release notes from the previous step.

    ```
    git push origin release/v$VERSION
    gh release create v$VERSION --latest --target release/v$VERSION --title v$VERSION --notes-file /tmp/yarpc-release-notes.txt
    ```

- Done, no need to merge the release branch back to main.
