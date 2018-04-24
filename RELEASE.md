Release process
===============

This document outlines how to create a release of yarpc-go

1.  Set up some environment variables for use later.

    ```
    # This is the version being released.
    VERSION=1.21.0

    # This is the branch from which $VERSION will be released.
    # This is almost always dev.
    BRANCH=dev
    ```

    ** If you are copying/pasting commands, make sure you actually set the right value for VERSION above. **

2.  Make sure you have the latest master.

    ```
    git checkout master
    git pull
    ```

3.  Merge the branch being released into master.

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
    make verifyversion
    ```

6.  Create a commit for the release.

    ```
    git add version.go CHANGELOG.md
    git commit -m "Preparing release v$VERSION"
    ```

7.  Tag and push the release.

    ```
    git tag -a "v$VERSION" -m "v$VERSION"
    git push origin master "v$VERSION"
    ```

8.  Go to <https://buildkite.com/uberopensource/yarpc-go/builds> and cancel the
    build for `v$VERSION`.  If that Codecov build completes before the Codecov
    build for master, the code coverage for master will not get updated because
    only one branch gets updated per commit; this was verified with Codecov
    support. This will get tested by the build for master anyways.

9.  Go to <https://github.com/yarpc/yarpc-go/tags> and edit the release notes
    of the new tag.  Copy the changelog entries for this release in the
    release notes and set the name of the release to the version number
    (`v$VERSION`).

10. Switch back to development.

    ```
    git checkout $BRANCH
    git merge master
    ```

11. Add a placeholder for the next version to CHANGELOG.md and a new link at
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

12. Update the version number in version.go to the same version.

    ```diff
    -const Version = "1.21.0"
    +const Version = "1.22.0-dev"
    ```

13. Verify the version number matches.

    ```
    make verifyversion
    ```

14. Commit and push your changes.

    ```
    git add CHANGELOG.md version.go
    git commit -m 'Back to development'
    git push origin $BRANCH
    ```
