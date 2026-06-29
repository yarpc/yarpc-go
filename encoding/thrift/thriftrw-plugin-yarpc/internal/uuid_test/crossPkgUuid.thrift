include "./crossPkgInner.thrift"

typedef crossPkgInner.CrossInner AliasedCrossInner

service TestService {
    string testMethod(
        1: AliasedCrossInner arg,
    )
}
