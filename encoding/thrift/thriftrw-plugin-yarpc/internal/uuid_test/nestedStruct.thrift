// Three-level struct chain: only the bottom field carries the
// annotation. The args walker must descend through every hop and
// surface a multi-step path (depth >= 3) ending at innerUUID.
struct InnerLevel {
    1: optional string innerUUID (auth.actor_uuid = "true")
}

struct MidLevel {
    1: optional InnerLevel inner
}

struct OuterLevel {
    1: optional MidLevel mid
}

service TestService {
    string testMethod(
        1: OuterLevel nested,
    )
}
