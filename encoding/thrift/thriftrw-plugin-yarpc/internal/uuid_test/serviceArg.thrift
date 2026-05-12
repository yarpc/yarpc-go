// Service-method parameters can carry the auth.actor_uuid annotation.
// thriftrw renders these into a synthetic <Service>_<Method>_Args struct,
// which is the type the yarpc plugin should attach ActorUUID() to.
service TestService {
    string testMethod(
        1: string notInterested,
        2: string interested (auth.actor_uuid = "true"),
    )
}
