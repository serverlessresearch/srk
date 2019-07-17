package cfbench


import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
	"log"
	"sync"
)

func genExperimentId() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return hex.EncodeToString(bytes)
}


type Progress struct {
	pending, running, completed, data int
	m sync.Mutex
}

func (p *Progress) setInvoked() {
	p.m.Lock()
	p.pending += 1
	p.m.Unlock()
}

func (p *Progress) setRunning() {
	p.m.Lock()
	p.pending -= 1
	p.running += 1
	p.m.Unlock()
}

func (p *Progress) setDone() {
	p.m.Lock()
	p.running -= 1
	p.completed += 1
	pending, running, completed := p.pending, p.running, p.completed
	p.m.Unlock()
	log.Printf("progress now [%d %d %d]", pending, running, completed)
}

func (p *Progress) setData() {
	p.m.Lock()
	p.data += 1
	p.m.Unlock()
}

func (p *Progress) allDone() bool {
	p.m.Lock()
	done := p.pending == 0 && p.running == 0 && p.completed != 0 && p.completed == p.data
	p.m.Unlock()
	return done
}



func invokeMulti(progress *Progress, n int) {
	sess := session.Must(session.NewSession())

	//sess := session.Must(session.NewSessionWithOptions(session.Options{
	//	SharedConfigState: session.SharedConfigEnable,
	//}))

	experimentId := genExperimentId()

	client := lambda.New(sess, &aws.Config{Region: aws.String("us-west-2")})

	for i := 0; i < n; i++ {
		payload, err := json.Marshal(map[string]interface{}{
			"uuid":          fmt.Sprintf("%s:%d", experimentId, i),
			"experimentId": experimentId,
			"sleep_time_ms": 3000,
			"tracking_url":  "http://192.168.60.101:3080/",
		})

		_, err = client.Invoke(&lambda.InvokeInput{FunctionName: aws.String("Hello"), Payload: payload, InvocationType: aws.String("Event") })
		if err != nil {
			log.Fatal("Error invoking function", err)
		}
		progress.setInvoked()
	}
	fmt.Printf("done")
}
