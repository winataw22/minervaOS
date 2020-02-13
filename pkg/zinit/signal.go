package zinit

// Signal is a type that maps linux signal to a string
// it is used by the Kill method
type Signal string

// List of supported signal
var (
	SIGABRT   = Signal("SIGABRT")
	SIGALRM   = Signal("SIGALRM")
	SIGBUS    = Signal("SIGBUS")
	SIGCHLD   = Signal("SIGCHLD")
	SIGCLD    = Signal("SIGCLD")
	SIGCONT   = Signal("SIGCONT")
	SIGFPE    = Signal("SIGFPE")
	SIGHUP    = Signal("SIGHUP")
	SIGILL    = Signal("SIGILL")
	SIGINT    = Signal("SIGINT")
	SIGIO     = Signal("SIGIO")
	SIGIOT    = Signal("SIGIOT")
	SIGKILL   = Signal("SIGKILL")
	SIGPIPE   = Signal("SIGPIPE")
	SIGPOLL   = Signal("SIGPOLL")
	SIGPROF   = Signal("SIGPROF")
	SIGPWR    = Signal("SIGPWR")
	SIGQUIT   = Signal("SIGQUIT")
	SIGSEGV   = Signal("SIGSEGV")
	SIGSTKFLT = Signal("SIGSTKFLT")
	SIGSTOP   = Signal("SIGSTOP")
	SIGSYS    = Signal("SIGSYS")
	SIGTERM   = Signal("SIGTERM")
	SIGTRAP   = Signal("SIGTRAP")
	SIGTSTP   = Signal("SIGTSTP")
	SIGTTIN   = Signal("SIGTTIN")
	SIGTTOU   = Signal("SIGTTOU")
	SIGUNUSED = Signal("SIGUNUSED")
	SIGURG    = Signal("SIGURG")
	SIGUSR1   = Signal("SIGUSR1")
	SIGUSR2   = Signal("SIGUSR2")
	SIGVTALRM = Signal("SIGVTALRM")
	SIGWINCH  = Signal("SIGWINCH")
	SIGXCPU   = Signal("SIGXCPU")
	SIGXFSZ   = Signal("SIGXFSZ")
)
