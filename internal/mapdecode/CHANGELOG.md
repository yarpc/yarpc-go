Releases
========

v0.2.0 (unreleased)
-------------------

-   Added `DecodeHook` to intercept values before they are decoded.
-   Added `FieldHook` to intercept values before they are decoded into specific
    struct fields.
-   Decode now parses strings if they are found in place of a float, boolean,
    or integer.
-   Embedded structs and maps will now be inlined into their parent structs.


v0.1.0 (2017-03-31)
-------------------

-   Initial release.
