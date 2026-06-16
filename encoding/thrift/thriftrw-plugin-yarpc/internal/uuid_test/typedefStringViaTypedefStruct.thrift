typedef string ActorIdentifier

struct Inner {
    1: optional ActorIdentifier id (auth.actor_uuid = "true")
}

typedef Inner AliasedInner

service TestService {
    string testMethod(
        1: AliasedInner arg,
    )
}
