package yarpcerrors

// WrapHandlerError is a convenience function to help wrap errors returned
// from a handler.
//
// If err is nil, WrapHandlerError returns nil.
// If err is a YARPC error, WrapHandlerError returns err with no changes.
// If err is not a YARPC error, WrapHandlerError returns a new YARPC error
// with code CodeUnknown and message err.Error(), along with
// service and procedure information.
func WrapHandlerError(err error, service string, procedure string) error {
	if err == nil {
		return nil
	}
	if IsStatus(err) {
		return err
	}
	return Newf(CodeUnknown, "error for service %q and procedure %q: %s", service, procedure, err.Error())
}
