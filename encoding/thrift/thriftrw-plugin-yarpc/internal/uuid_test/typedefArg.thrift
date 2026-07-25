// Service-method parameter whose type is a typedef of string.
// thriftrw's generated getter then returns the typedef rather than
// plain string, so the path walker must mark the leaf as IsTypedef
// and the template emits a string(...) cast around the chain to
// satisfy the ActorUUID() return type.
typedef string ActorIdentifier

service TestService {
    string testMethod(
        1: ActorIdentifier identifier (auth.actor_uuid = "true"),
    )
}
