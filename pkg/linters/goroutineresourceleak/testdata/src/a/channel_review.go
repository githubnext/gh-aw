package a

func channelAliasClosed() {
	go func() {
		ch := make(chan int)
		alias := ch
		close(alias)
	}()
}

func channelAliasOperations() {
	go func() {
		ch := make(chan int, 1)
		alias := ch
		alias <- 1
		_ = <-alias
	}()
}

func channelAliasAssigned() {
	go func() {
		ch := make(chan int)
		var alias chan int
		alias = ch
		close(alias)
	}()
	go func() {
		ch := make(chan int)
		var alias = ch
		close(alias)
	}()
}

func consumeReviewChannels(channels []chan int) {}

func channelAppended() {
	go func() {
		ch := make(chan int)
		var pending []chan int
		pending = append(pending, ch)
		consumeReviewChannels(pending)
	}()
	go func() {
		ch := make(chan int)
		consumeReviewChannels(append([]chan int(nil), ch))
	}()
}

func channelConvertedReceive() {
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		_ = <-(<-chan int)(ch)
	}()
	go func() {
		ch := make(chan int, 1)
		(chan<- int)(ch) <- 1
		_ = <-(<-chan int)(ch)
	}()
	go func() {
		ch := make(chan int)
		close((chan<- int)(ch))
	}()
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		_ = (<-chan int)(ch)
		ch <- 1
	}()
}

func channelReturnedAfterSend() {
	go func() chan int { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		ch <- 1
		return ch
	}()
}

func observeReviewBool(value bool) {}

func channelCompared() {
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		observeReviewBool(ch == nil)
		ch <- 1
	}()
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		observeReviewBool(nil != ch)
		ch <- 1
	}()
}

func handoffReviewChannel(ch chan int) chan int { return ch }

func channelHandoffInsideComparison() {
	go func() {
		ch := make(chan int)
		observeReviewBool(handoffReviewChannel(ch) == nil)
	}()
}

func channelBuiltinSize() {
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		doSomeWorkWithID(cap(ch))
	}()
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		values := append([]int(nil), len(ch), cap(ch))
		_ = values
	}()
}

func channelDiscardIsNotAlias() {
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		_ = ch
	}()
	go func() { // want "goroutine creates channel without safe buffered send/receive pairing or close"
		ch := make(chan int)
		_, value := ch, 1
		_ = value
	}()
}
