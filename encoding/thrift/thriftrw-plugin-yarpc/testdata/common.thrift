service EmptyService {}

service ExtendEmpty extends EmptyService {
    void hello()
}

service BaseService {
    bool healthy()
}
