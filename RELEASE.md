# Release process

This document outlines how to create a release of yarpc-go

1. `git checkout master`

2. `git pull`

3. `git merge <branch>` where `<branch>` is the branch we want to cut the release on (most likely `dev`)

4. Alter CHANGELOG.md from `v<version>-dev (unreleased)` to `v<version_to_release> (YYYY-MM-DD)`

5. Alter version.go to have the same version as `version_to_release`

6. Create a commit with the title `Preparing for release <version_to_release>`

7. Create a git tag for the version using `git tag -a v<version_to_release> -m v<version_to_release` (e.g. `git tag -a v1.0.0 -m v1.0.0`)

8. Push the tag to origin `git push --tags origin v<version_to_release>`

9. `git push origin master`

10. Go to https://github.com/yarpc/yarpc-go/tags and edit the release notes of the new tag (copy the changelog into the release notes and make the release name the version number)

11. `git checkout dev`

12. `git merge master`

13. Update CHANGELOG.md and version.go to have a new `v<version>-dev (unreleased)` and put into a commit with title `Back to development`

14. `git push origin dev`