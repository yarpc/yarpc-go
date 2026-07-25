// Container of every shape of misplaced or unsupported annotation.

typedef string ActorIdentifier
typedef string IgnoredOnTypedef (auth.actor_uuid = "true")
typedef ActorIdentifier NestedActor

struct IgnoredCases {
    1: optional i64 timestamp (auth.actor_uuid = "true")
    2: optional NestedActor nested (auth.actor_uuid = "true")
} (auth.actor_uuid = "true")

service TestService {
    string testMethod(
        1: i64 stamp (auth.actor_uuid = "true"),
        2: NestedActor identifier (auth.actor_uuid = "true"),
    )
}
