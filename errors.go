package sunrpc

import "fmt"

type ErrRpcMismatch struct {
	High, Low uint32
}

func (e *ErrRpcMismatch) Error() string {
	return fmt.Sprintf("RPC version mismatch, found: %v.%v", e.High, e.Low)
}

type ErrAuth struct {
	Stat AuthStat
}

func (e *ErrAuth) Error() string {
	return fmt.Sprintf("RPC auth unsupported, found: %v", e.Stat)
}

type ErrProgMismatch struct {
	High, Low uint32
}

func (e *ErrProgMismatch) Error() string {
	return fmt.Sprintf("program mismatch, found: %v.%v", e.High, e.Low)
}

type ErrProgUnavail struct{}
type ErrProcUnavail struct{}
type ErrGarbageArgs struct{}

func (e *ErrProgUnavail) Error() string { return "requested program unavailable" }
func (e *ErrProcUnavail) Error() string { return "requested procedure unavailable" }
func (e *ErrGarbageArgs) Error() string { return "garbage arguments for proc" }
