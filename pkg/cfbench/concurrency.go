package cfbench

import (
	"sort"
	"time"
)

type LaunchMessage struct {
	Duration    time.Duration
	ReferenceId int
}

type CompletionMessage struct {
	ReferenceId int
}

type TransitionPoint struct {
	concurrency int
	when        time.Duration
}

type ConcurrencySpan struct {
	concurrency int
	begin       time.Duration
	end         time.Duration
}

type ConcurrencyControl struct {
	concurrencySpans  []ConcurrencySpan
	launchChannel     chan<- LaunchMessage
	completionChannel <-chan CompletionMessage
}

func NewConcurrencyControl(concurrencySpans []ConcurrencySpan, launchChannel chan<- LaunchMessage, completionChannel <-chan CompletionMessage) (*ConcurrencyControl, error) {
	//if len(concurrencySpans) < 2 {
	//	return nil, errors.New("must provide at least two transition points")
	//}
	//prevStart := concurrencySpans[0].when
	//for _, t := range concurrencySpans[1:] {
	//	if prevStart >= t.when {
	//		return nil, errors.New("non-monotonic transition points")
	//	}
	//	prevStart = t.when
	//}
	return &ConcurrencyControl{concurrencySpans, launchChannel, completionChannel}, nil
}

type spanChange struct {
	when   time.Duration
	spanId int
	active bool
}

func (cc *ConcurrencyControl) Run() {
	activeSpanEnds := make(map[int]time.Duration)
	var spanChanges []spanChange
	for i, s := range cc.concurrencySpans {
		spanChanges = append(spanChanges, spanChange{s.begin, i, true})
		spanChanges = append(spanChanges, spanChange{s.end, i, false})
	}
	sort.Slice(spanChanges, func(i, j int) bool { return spanChanges[i].when < spanChanges[j].when })
	var spanChangeIndex = 0
	var startTime = time.Now()
	var spanChangeTimer = time.NewTimer(spanChanges[spanChangeIndex].when)
	for ; spanChangeIndex < len(spanChanges); {
		select {
		case cm := <-cc.completionChannel:
			endTime, found := activeSpanEnds[cm.ReferenceId]
			if found {
				elapsed := time.Now().Sub(startTime)
				if elapsed < endTime {
					cc.launchChannel <- LaunchMessage{endTime - elapsed, cm.ReferenceId}
				}
			}
		case <-spanChangeTimer.C:
			spanChange := spanChanges[spanChangeIndex]
			elapsed := time.Now().Sub(startTime)
			if spanChange.active {
				span := cc.concurrencySpans[spanChange.spanId]
				maxDuration := span.end - elapsed
				for i := 0; i < span.concurrency; i++ {
					cc.launchChannel <- LaunchMessage{maxDuration, spanChange.spanId}
				}
				activeSpanEnds[spanChange.spanId] = span.end
			} else {
				delete(activeSpanEnds, spanChange.spanId)
			}
			spanChangeIndex += 1
			if spanChangeIndex < len(spanChanges) {
				spanChangeTimer = time.NewTimer(spanChanges[spanChangeIndex].when - elapsed)
			}
		}
	}
}
