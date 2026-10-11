package combined

import (
	"errors"
	"io"
	"os"
)

func uncheckedFile(filename string) error {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close() // want `unchecked_deferredclose: defer close\(\) call ignores error return value`
	return nil
}

type customCloser struct{}

func (customCloser) Close() error { return nil }

func uncheckedCustom(c customCloser) {
	defer c.Close() // want `unchecked_deferredclose: defer close\(\) call ignores error return value`
}

func uncheckedInterface(c io.Closer) {
	c.Close()       // want `closeerrorunchecked: Close\(\) error is explicitly discarded`
	_ = c.Close()   // want `closeerrorunchecked: Close\(\) error is explicitly discarded`
	defer c.Close() // want `unchecked_deferredclose: defer close\(\) call ignores error return value`
}

func uncheckedClosure(c io.Closer) {
	defer func() {
		c.Close() // want `closeerrorunchecked: Close\(\) error is explicitly discarded`
	}()
	defer func() {
		_ = c.Close() // want `closeerrorunchecked: Close\(\) error is explicitly discarded`
	}()
}

func checkedFile(filename string) (err error) {
	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()
	return nil
}

func checkedCustom(c customCloser) (err error) {
	defer func() {
		err = errors.Join(err, c.Close())
	}()
	return nil
}

func checkedInterface(c io.Closer) (err error) {
	defer func() {
		if closeErr := c.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()
	return nil
}

type noErrorCloser struct{}

func (noErrorCloser) Close() {}

func noError(c noErrorCloser) {
	c.Close()
	defer c.Close()
}

func targetedSuppressions(c io.Closer) {
	c.Close()       //nolint:closeerrorunchecked
	defer c.Close() //nolint:unchecked_deferredclose
	//nolint:closeerrorunchecked
	defer c.Close() // want `unchecked_deferredclose: defer close\(\) call ignores error return value`
	//nolint:unchecked_deferredclose
	c.Close() // want `closeerrorunchecked: Close\(\) error is explicitly discarded`
}
