service EmptyService {}

service ExtendEmpty extends EmptyService {
    void hello()
}

service BaseService {
    bool healthy()
}

service ExtendOnly extends BaseService {
    // A service without any functions except inherited ones
}
