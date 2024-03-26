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
    VERSION=1.21.0

    # This is the branch from which $VERSION will be released.
    # This is almost always dev.
    BRANCH=dev
    ```

    **If you are copying/pasting commands, make sure you actually set the right
    value for VERSION above.**

2. Call release preparation helper.

   ```
   ./etc/bin/release.sh $VERSION $BRANCH
   ```

3. Check for the diff in the CHANGELOG.md and version.go and make sure it looks good.

    ```
    git diff CHANGELOG.md version.go
    ```

4.  Create a commit for the release.

    ```
    git add version.go CHANGELOG.md
    git commit -m "Preparing release v$VERSION"
    ```

5.  Make a pull request with these changes against `master`.

    ```
    gh pr create --base master --title "Preparing release v$VERSION" --web
    ```

6.  Land the pull request after approval as a **merge commit**. To do this,
    select **Create a merge commit** from the pull-down next to the merge
    button and click **Merge pull request**. Make sure you delete that branch
    after it has been merged with **Delete Branch**.

7.  Once the change has been landed, pull it locally.

    ```
    git checkout master
    git pull
    ```

8. Tag a release.

    ```
    gh release create v$VERSION --latest --target master --title "v$VERSION"
    ```

9. Copy the changelog entries for this release into the release description in
    the newly opened browser window.

10. Switch back to development.

    ```
    git checkout $BRANCH
    git merge master
    ```
    
11. Run helper script to update dev branch, CHANGELOG.md and version.go.

    ```
    ./etc/bin/back-to-development.sh $VERSION $BRANCH
    ```

12. Verify git log and changes.

    ```
    git log --oneline -n 5
    git diff CHANGELOG.md version.go
    ```

13. Commit and push your changes.

    ```
    git add CHANGELOG.md version.go
    git commit -m 'Back to development'
    git push origin $BRANCH
    ```
