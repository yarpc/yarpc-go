// Two directly-annotated primitive args in the same method. The
// generated *_Args type can only carry one ActorUUID() accessor, so
// validateUUIDAnnotations must reject this before the template runs.
service TestService {
    string testMethod(
        1: string firstUUID  (auth.actor_uuid = "true"),
        2: string secondUUID (auth.actor_uuid = "true"),
    )
}
