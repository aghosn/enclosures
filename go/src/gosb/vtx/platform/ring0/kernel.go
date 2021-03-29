package ring0

// Init initializes a new kernel.
//
// N.B. that constraints on KernelOpts must be satisfied.
//
//go:nosplit
func (k *Kernel) Init(opts KernelOpts) {
	k.init(opts)
}

// Halt halts execution.
func Halt()

// defaultHooks implements hooks.
type defaultHooks struct{}

// KernelSyscall implements Hooks.KernelSyscall.
//
//go:nosplit
func (defaultHooks) KernelSyscall() { Halt() }

// KernelException implements Hooks.KernelException.
//
//go:nosplit
func (defaultHooks) KernelException(Vector) { Halt() }

// kernelSyscall is a trampoline.
//
//go:nosplit
func kernelSyscall(c *CPU) { c.hooks.KernelSyscall() }

// kernelException is a trampoline.
//
//go:nosplit
func kernelException(c *CPU, vector Vector) { c.hooks.KernelException(vector) }

// Init initializes a new CPU.
//
// Init allows embedding in other objects.
func (c *CPU) Init(k *Kernel, hooks Hooks) {
	c.self = c   // Set self reference.
	c.kernel = k // Set kernel reference.
	c.init()     // Perform architectural init.

	// Require hooks.
	if hooks != nil {
		c.hooks = hooks
	} else {
		c.hooks = defaultHooks{}
	}
}
