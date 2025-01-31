Release process
===============

> **NOTE**: Don't do any of this before validating with a test service. Check
> the internal release doc (ask someone about it) for more information.

This document outlines how to create a release of yarpc-go.

Prerequisites
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

Releasing
---------

1.  Set up some environment variables for use later.

    ```
    # This is the version being released.
    VERSION=1.0.0

    # This is the branch from which $VERSION will be released.
    # This is almost always dev.
    BRANCH=dev
    ```

    **If you are copying/pasting commands, make sure you actually set the right
    value for VERSION above.**

2. Call release preparation helper.

   ```
   ./etc/bin/release-step-1.sh $VERSION $BRANCH
   ```
   
    This script will:
    * Create a release branch from the specified branch.
    * Update CHANGELOG.md with the new version and date.
    * Update version.go with the new version.
    * Commit the changes and push the branch to GitHub.
    * Open a pull request for the release.

3.  Land the pull request after approval as a **merge commit**. To do this,
    select **Create a merge commit** from the pull-down next to the merge
    button and click **Merge pull request**. Make sure you delete that branch
    after it has been merged with **Delete Branch**.

4.  Once the change has been landed, run second script.

   ```
   ./etc/bin/release-step-2.sh $VERSION $BRANCH
   ```

   This script will:
   * Tag the release.
   * Push the tag to GitHub.
   * Create a release on GitHub.
      * (This will open a browser window with the release page. Copy the changelog entries into the release description.)
   * Switch version and CHANGELOG.md back to development mode.
   * Commit the changes and push them to GitHub.
   * Open a pull request for the development changes.

5. Send pull request to a peer review.

# Manual release

If the above steps fail, you can manually release yarpc-go by following the
steps below.

(For the sake of simplicity, we will assume that changes are merged to branch `dev`,
release should be done to branch `master`, and new version is 1.0.0.)

1.  Create a release branch from `dev`.

    ```
    git checkout dev
    git pull
    git checkout -B prepare-release
    ```
    
2.  Update `CHANGELOG.md` with the new version and date.

    ```
    - ## [Unreleased]
    + ## [1.0.0] - 2025-01-01
    
    - [Unreleased]: https://github.com/yarpc/yarpc-go/compare/v0.0.9...HEAD
    + [1.75.4]: https://github.com/yarpc/yarpc-go/compare/v0.0.9...v1.0.0
    ```
    
3.  Update `version.go` with the new version.

    ```
    - const Version = "0.0.9"
    + const Version = "1.0.0"
    ```
    
4.  Commit the changes and push the branch to GitHub.

    ```
    git add CHANGELOG.md version.go
    git commit -m "Preparing release v1.0.0"
    gh pr create --base master --title "Preparing release v1.0.0" --web
    ```

5.  Land the pull request after approval as a **merge commit**.

6.  After the change has been landed, pull the changes locally and tag the release.

    ```
    git checkout master
    git pull
    gh release create "v1.0.0" --latest --target master --title "v1.0.0"
    ```
    
Use the changelog entries as the release description.

7. Merging release back to dev branch via pull request.

    ```
    git checkout dev
    git pull
    git checkout -B return-to-development
    git merge origin/master
    ```

8.  Switch version and `CHANGELOG.md` back to development mode in a new branch.

    ```
    + ## [Unreleased]
    + - No changes yet.
    
    + [Unreleased]: https://github.com/yarpc/yarpc-go/compare/v1.0.0...HEAD
    ```
    
9. Commit the changes and push the branch to GitHub.

    ```
    git add CHANGELOG.md version.go
    git commit -m "Return to development"
    gh pr create --base dev --title "Return to development" --web
    ```

10. Land the pull request after approval **without** merge commit.
