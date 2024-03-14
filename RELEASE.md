Release process
===============

> **NOTE**: Don't do any of this before validating with a test service. Check
> the internal release doc (ask someone about it) for more information.

This document outlines how to create a release of yarpc-go.

Prerequisites
-------------

Make sure you have `hub` installed.

```
brew install hub
```

Releasing
---------

1.  Set up some environment variables for use later.

    ```
    # This is the version being released.
    VERSION=1.21.0

    # This is the branch from which $VERSION will be released.
    # This is almost always dev.
    BRANCH=dev
    ```

    **If you are copying/pasting commands, make sure you actually set the right
    value for VERSION above.**

2.  Make sure you have the latest master and create a new release branch off of
    it.

    ```
    git fetch origin master
    git checkout origin/master
    git checkout -B $(whoami)/release
    ```

3.  Merge the branch with the changes being released into the newly created
    release branch.

    ```
    git merge $BRANCH
    ```

4.  Alter the Unreleased entry in CHANGELOG.md to point to `$VERSION` and
    update the link at the bottom of the file. Use the format `YYYY-MM-DD` for
    the year.

    ```diff
    -## [Unreleased]
    +## [1.21.0] - 2017-10-23
    ```

    ```diff
    -[Unreleased]: https://github.com/yarpc/yarpc-go/compare/v1.20.1...HEAD
    +[1.21.0]: https://github.com/yarpc/yarpc-go/compare/v1.20.1...v1.21.0
    ```


5.  Update the version number in version.go and verify that it matches what is
    in the changelog.

    ```
    sed -i '' -e "s/^const Version =.*/const Version = \"$VERSION\"/" version.go
    SUPPRESS_DOCKER=1 make verifyversion
    ```

6.  Create a commit for the release.

    ```
    git add version.go CHANGELOG.md
    git commit -m "Preparing release v$VERSION"
    ```

7.  Make a pull request with these changes against `master`.

    ```
    git push && hub pull-request --push --browse -b master
    ```

8.  Land the pull request after approval as a **merge commit**. To do this,
    select **Create a merge commit** from the pull-down next to the merge
    button and click **Merge pull request**. Make sure you delete that branch
    after it has been merged with **Delete Branch**.

9.  Once the change has been landed, pull it locally.

    ```
    git checkout master
    git pull
    ```

10. Tag a release.

    ```
    hub release create -o -m v$VERSION -t master v$VERSION
    ```

11. Copy the changelog entries for this release into the release description in
    the newly opened browser window.

12. Go to <https://buildkite.com/uberopensource/yarpc-go/builds> and cancel the
    build for `v$VERSION`. If that Codecov build completes before the Codecov
    build for master, the code coverage for master will not get updated because
    only one branch gets updated per commit; this was verified with Codecov
    support. This will get tested by the build for master anyways.

13. Switch back to development.

    ```
    git checkout $BRANCH
    git merge master
    ```

14. Add a placeholder for the next version to CHANGELOG.md and a new link at
    the bottom.

    ```diff
    +## [Unreleased]
    +- No changes yet.
    +
     ## [1.21.0] - 2017-10-23
    ```

    ```diff
    +[Unreleased]: https://github.com/yarpc/yarpc-go/compare/v1.21.0...HEAD
     [1.21.0]: https://github.com/yarpc/yarpc-go/compare/v1.20.1...v1.21.0
    ```

15. Update the version number in version.go to the same version.

    ```diff
    -const Version = "1.21.0"
    +const Version = "1.22.0-dev"
    ```

16. Verify the version number matches.

    ```
    make verifyversion SUPPRESS_DOCKER=1
    ```

17. Commit and push your changes.

    ```
    git add CHANGELOG.md version.go
    git commit -m 'Back to development'
    git push origin $BRANCH
    ```
