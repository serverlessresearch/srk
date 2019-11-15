package cfbench

import (
	"github.com/google/uuid"
	"log"
	"sort"
	"time"
)

type LaunchMessage struct {
	Duration    time.Duration
	ReferenceId string
}

type CompletionMessage struct {
	ReferenceId string
	Success     bool
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

	var running = make(map[string]int)

	launchNew := func(maxDuration time.Duration, spanId int) {
		id, err := uuid.NewRandom()
		if err != nil {
			log.Fatal(err)
		}
		idStr := id.String()
		running[idStr] = spanId
		cc.launchChannel <- LaunchMessage{maxDuration, idStr}
	}

	for spanChangeIndex < len(spanChanges) || len(running) > 0 {
		log.Printf("looping, index is %d", spanChangeIndex)
		select {
		case cm := <-cc.completionChannel:
			if spanId, ok := running[cm.ReferenceId]; ok {
				delete(running, cm.ReferenceId)
				endTime, spanActive := activeSpanEnds[spanId]
				if spanActive {
					elapsed := time.Now().Sub(startTime)
					if elapsed < endTime {
						launchNew(endTime-elapsed, spanId)
					}
				}
			}
		case <-spanChangeTimer.C:
			spanChange := spanChanges[spanChangeIndex]
			log.Printf("span change %+v", spanChange)
			elapsed := time.Now().Sub(startTime)
			if spanChange.active {
				span := cc.concurrencySpans[spanChange.spanId]
				maxDuration := span.end - elapsed
				log.Printf("number to launch is %d\n", span.concurrency)
				for i := 0; i < span.concurrency; i++ {
					launchNew(maxDuration, spanChange.spanId)
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
	log.Printf("scan done\n")
}
