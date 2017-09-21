Release process
===============

This document outlines how to create a release of yarpc-go

1.  `git checkout master`

2.  `git pull`

3.  `git merge <branch>` where `<branch>` is the branch we want to cut the
    release on (most likely `dev`)

4.  Alter CHANGELOG.md from `v<version>-dev (unreleased)` to
    `v<version_to_release> (YYYY-MM-DD)`

5.  Alter `version.go` to have the same version as `version_to_release`

6.  Run `make verifyversion`

7.  Create a commit with the title `Preparing for release <version_to_release>`

8.  Create a git tag for the version using
    `git tag -a v<version_to_release> -m v<version_to_release` (e.g.
    `git tag -a v1.0.0 -m v1.0.0`)

9.  Push the tag to origin `git push --tags origin v<version_to_release>`

10. `git push origin master`

11. Go to https://travis-ci.org/yarpc/yarpc-go/builds and cancel the build for
    v<version_to_release>. If the Codecov build for v<version_to_release>
    completes before the Codecov build for master, the code coverage for master
    will not get updated, only one branch gets updated per commit, this is
    verified with Codecov support. This will get tested by the build for master
    anyways.

12. Go to https://github.com/yarpc/yarpc-go/tags and edit the release notes of
    the new tag (copy the changelog into the release notes and make the release
    name the version number)

13. `git checkout dev`

14. `git merge master`

15. Update `CHANGELOG.md` and `version.go` to have a new
    `v<version>-dev (unreleased)`

16. Run `make verifyversion`

17. Create a commit with the title `Back to development`

18. `git push origin dev`
