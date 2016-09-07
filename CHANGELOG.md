Releases
========

v0.1.1 (2016-09-01)
-------------------

-   Use `github.com/yarpc/yarpc-go` as the import path; revert use of
    `go.uber.org/yarpc` vanity path. There is an issue in Glide `0.11` which
    causes installing these packages to fail, and thriftrw `~0.1`'s yarpc
    template is still using `github.com/yarpc/yarpc-go`.


v0.1.0 (2016-08-31)
-------------------

-   Initial release.
