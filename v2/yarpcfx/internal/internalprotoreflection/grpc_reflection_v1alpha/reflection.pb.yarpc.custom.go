package grpc_reflection_v1alpha

// YARPCReflectionFileDescriptors returns an array of encoded filedescriptor
// for yaprc to use. Normally the filedescriptors are accessed through the
// reflection.ServerMeta that is injected into the container. The injection is
// happens using New{}YARPCProcedures to automatically inject for all services
// using the fx pattern.
// This will not work for the reflection service due to a chicken and egg
// problem: we need a server to access the reflection.Meta and to create a
// server we need access to all the reflection.Meta (including our own).
// We could use throwaway instantiation of the reflection service to access
// its meta, but this would require using the interface{} response of
// New{}YARPCProcedures meant for fx. Instead of being type unsafe, here we
// augment the generated code to get compile time safe access to the required
// filedescriptor
//
// After regeneration with the yarpc plugin, update this reference
var YARPCReflectionFileDescriptors = yarpcFileDescriptorClosure42a8ac412db3cb03
