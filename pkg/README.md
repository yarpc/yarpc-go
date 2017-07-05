pkg/ is a collection of utility packages used by the yarpc-go project.

Utility packages are kept separate from the yarpc-go core codebase to keep it
as small and concise as possible. If some utilities grow larger and their APIs
stabilize, they may be moved to their own repository under the yarpc or uber-go
organizations, to facilitate re-use by other projects. However that is not the
priority.

There is no guarantee of API stability for packages within pkg/.
