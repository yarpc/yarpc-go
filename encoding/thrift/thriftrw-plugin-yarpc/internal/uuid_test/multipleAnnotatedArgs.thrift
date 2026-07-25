// Method-args list reaches the auth.actor_uuid annotation through
// several args.

struct Inner {
    1: required string id (auth.actor_uuid = "true")
}

service TestService {
    string testMethod(
        1: string firstUUID  (auth.actor_uuid = "true"),
        2: Inner  second,
        3: string thirdUUID  (auth.actor_uuid = "true"),
    )
}
