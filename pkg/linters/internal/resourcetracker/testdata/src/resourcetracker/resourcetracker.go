package resourcetracker

type resource struct{}

func acquire() *resource { return &resource{} }

func (r *resource) Release() error { return nil }

func ManualRelease() {
	r := acquire() // want `resource should be released with defer`
	r.Release()
}

func DeferredRelease() {
	r := acquire()
	defer r.Release()
}

func AcquiredAndLeaked() {
	r := acquire()
	_ = r
}

func WrapperClosureDeferredRelease() {
	r := acquire()
	defer func() {
		r.Release()
	}()
}

func ReleaseResultAssigned() {
	r := acquire() // want `resource should be released with defer`
	err := r.Release()
	_ = err
}

func ClosureBodyNotTracked() {
	r := acquire()
	defer r.Release()

	fn := func() {
		// Closure bodies are independent execution contexts; no diagnostic expected.
		inner := acquire()
		inner.Release()
	}
	fn()
}

func ReassignReportsPreviousViolation() {
	r := acquire() // want `resource should be released with defer`
	r.Release()

	r = acquire()
	defer r.Release()
}

func NoLintSuppresses() {
	r := acquire() //nolint:resourcetrackertest
	r.Release()
}

func Shadowing() {
	r := acquire()
	defer r.Release()
	{
		r := acquire() // want `resource should be released with defer`
		r.Release()
	}
}
