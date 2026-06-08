// Service-method parameter whose type is a typedef of string. thriftrw's
// generated getter then returns the typedef rather than plain string, so
// uuidFieldInArgs must set .IsTypedef=true to make the template emit a
// string(...) cast and satisfy the ActorUUID() string return type.
typedef string ActorIdentifier

service TestService {
    string testMethod(
        1: ActorIdentifier identifier (auth.actor_uuid = "true"),
    )
}
