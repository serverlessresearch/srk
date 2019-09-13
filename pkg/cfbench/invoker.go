package cfbench

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

func genExperimentId() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}

type stringSet map[string]struct{}

var member struct{}

func (s *stringSet) add(key string) {
	(*s)[key] = member
}

func (s *stringSet) remove(key string) {
	delete(*s, key)
}

func (s *stringSet) contains(key string) bool {
	_, exists := (*s)[key]
	return exists
}

func (s *stringSet) size() int {
	return len(*s)
}

type progress struct {
	updateNotice                                  chan bool
	experimentId                                  string
	seqId                                         int
	pendingSet, runningSet, completedSet, dataSet stringSet
	invocationDone                                bool
	m                                             sync.Mutex
}

func newProgress(experimentId string) progress {
	p := progress{}
	p.updateNotice = make(chan bool)
	p.experimentId = experimentId
	p.seqId = 1
	p.pendingSet = make(stringSet)
	p.runningSet = make(stringSet)
	p.completedSet = make(stringSet)
	p.dataSet = make(stringSet)
	return p
}

func (p *progress) nextInvocationSeq() int {
	p.m.Lock()
	id := p.seqId
	p.seqId += 1
	p.m.Unlock()
	return id
}

func (p *progress) setInvoked(uuid string) {
	p.m.Lock()
	if !strings.HasPrefix(uuid, p.experimentId) {
		panic("invalid invocation")
	}
	if !p.pendingSet.contains(uuid) {
		p.pendingSet.add(uuid)
	}
	p.m.Unlock()
}

func (p *progress) setInovcationDone() {
	p.m.Lock()
	p.invocationDone = true
	p.m.Unlock()
}

func (p *progress) setRunning(uuid string) {
	p.m.Lock()
	if strings.HasPrefix(uuid, p.experimentId) && p.pendingSet.contains(uuid) {
		p.pendingSet.remove(uuid)
		p.runningSet.add(uuid)
	}
	p.m.Unlock()
	p.updateNotice <- true
}

func (p *progress) setDone(uuid string) {
	p.m.Lock()
	if strings.HasPrefix(uuid, p.experimentId) && p.runningSet.contains(uuid) {
		p.runningSet.remove(uuid)
		p.completedSet.add(uuid)
	}
	pending, running, completed, data := p.pendingSet.size(), p.runningSet.size(), p.completedSet.size(), p.dataSet.size()
	invocationDone := p.invocationDone
	p.m.Unlock()
	log.Printf("progress now [%d %d %d %d %t]", pending, running, completed, data, invocationDone)
	p.updateNotice <- true
}

func (p *progress) setData(uuid string) {
	p.m.Lock()
	if strings.HasPrefix(uuid, p.experimentId) && !p.dataSet.contains(uuid) {
		p.dataSet.add(uuid)
	}
	p.m.Unlock()
}

func (p *progress) allDone() bool {
	p.m.Lock()
	done := p.invocationDone && p.pendingSet.size() == 0 && p.runningSet.size() == 0 && p.completedSet.size() != 0 &&
		p.completedSet.size() == p.dataSet.size()
	p.m.Unlock()
	//log.Printf("Done check found %t", done)
	return done
}

func (p *progress) getConcurrency() int {
	p.m.Lock()
	concurrency := p.pendingSet.size() + p.runningSet.size()
	p.m.Unlock()
	return concurrency
}

func invokeMulti(experimentId string, trackingUrl string, functionName string, functionArgs map[string]interface{}, sweepDefinition *[]TransitionPoint, progress *progress) {
	sess := session.Must(session.NewSession())
	client := lambda.New(sess, &aws.Config{Region: aws.String("us-west-2")})

	invoke := func(n int) {
		for i := 0; i < n; i++ {
			invocationId := progress.nextInvocationSeq()
			uuid := fmt.Sprintf("%s:%d", experimentId, invocationId)
			args := map[string]interface{}{
				"uuid":         uuid,
				"experimentId": experimentId,
				"tracking_url": trackingUrl,
			}
			for k, v := range functionArgs {
				if _, exists := args[k]; !exists {
					args[k] = v
				} else {
					panic("argument conflict")
				}
			}
			payload, err := json.Marshal(args)

			_, err = client.Invoke(&lambda.InvokeInput{FunctionName: aws.String(functionName), Payload: payload, InvocationType: aws.String("Event")})
			if err != nil {
				log.Fatal("Error invoking function", err)
			}
			progress.setInvoked(uuid)
		}
	}

	go func() {
		nextIndex := 0
		var targetConcurrency int
		var startTime = time.Now()
		timer := time.NewTimer((*sweepDefinition)[nextIndex].when)
		for {
			select {
			case <-progress.updateNotice:
				launched := progress.getConcurrency()
				if launched < targetConcurrency {
					invoke(targetConcurrency - launched)
				}
			case <-timer.C:
				targetConcurrency = (*sweepDefinition)[nextIndex].concurrency
				log.Printf("update concurrency to %d\n", targetConcurrency)
				nextIndex += 1
				if nextIndex < len(*sweepDefinition) {
					timer.Reset((*sweepDefinition)[nextIndex].when - (time.Now().Sub(startTime)))
				}
				launched := progress.getConcurrency()
				if launched < targetConcurrency {
					invoke(targetConcurrency - launched)
				}
				if nextIndex >= len(*sweepDefinition) {
					progress.setInovcationDone()
				}
			}
		}
	}()
}
