package lambdalike

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/pkg/errors"
)

var logDebug = true

func debug(v ...interface{}) {
	if logDebug {
		log.Println(v...)
	}
}

func systemLog(msg string) {
	fmt.Fprintln(os.Stderr, "\033[32m"+msg+"\033[0m")
}

type ApiService struct {
	workerIPs          []string
	port               int
	Addr               string
	localWorkerManager *WorkerManager
	workChan           chan *requestContext
	functionsByName    map[string]*FunctionConfiguration
	functions          []FunctionConfiguration
	activeRequests     map[string]*requestContext
	activeRequestsLock sync.RWMutex
}

func NewApiService(workerIPs []string, port int) *ApiService {
	return &ApiService{
		workerIPs:       workerIPs,
		port:            port,
		Addr:            "",
		workChan:        make(chan *requestContext, 20),
		functionsByName: make(map[string]*FunctionConfiguration),
		activeRequests:  make(map[string]*requestContext),
	}
}

func (s *ApiService) Start() error {

	var addr string = "localhost:"
	if s.port > 0 {
		addr = fmt.Sprintf(":%d", s.port)
	}
	runtimeListener, err := net.Listen("tcp", addr)
	if err != nil {
		return errors.Wrap(err, "error opening listener")
	}
	s.Addr = runtimeListener.Addr().String()
	log.Printf("listening at %s", s.Addr)

	if len(s.workerIPs) == 0 {
		// start up a local WorkerManager
		portStr := s.Addr[strings.LastIndex(s.Addr, ":")+1:]
		log.Printf("port is at %s from %s", portStr, s.Addr)
		// host.docker.internal doesn't presently work on Linux - probably use 172.17.0.1
		s.localWorkerManager = NewWorkerManager(fmt.Sprintf("host.docker.internal:%s", portStr), nil)
	} else {
		log.Fatal("unimplemented")
	}

	var runtimeServer *http.Server

	runtimeRouter := s.createRuntimeRouter()
	runtimeServer = &http.Server{Handler: s.addAPIRoutes(runtimeRouter)}

	go runtimeServer.Serve(runtimeListener)

	return nil
}

func (s *ApiService) Shutdown() {
	if s.localWorkerManager != nil {
		s.localWorkerManager.Shutdown()
	}
}

func (s *ApiService) InstallFunction(fc FunctionConfiguration) error {
	if _, exists := s.functionsByName[fc.FnName]; exists {
		return errors.Errorf("function %s already installed", fc.FnName)
	}
	s.functions = append(s.functions, fc)
	s.functionsByName[fc.FnName] = &s.functions[len(s.functions)-1]

	s.localWorkerManager.Configure(s.functions)
	return nil
}

func (s *ApiService) contextInvoke(context *requestContext) {
	s.activeRequestsLock.Lock()
	defer s.activeRequestsLock.Unlock()

	s.activeRequests[context.RequestID] = context
	s.workChan <- context
}

func (s *ApiService) contextLookup(requestID string) (*requestContext, bool) {
	s.activeRequestsLock.RLock()
	defer s.activeRequestsLock.RUnlock()

	context, found := s.activeRequests[requestID]
	return context, found
}

func (s *ApiService) contextCompleted(context *requestContext) {
	log.Printf("setting context to completed")
	s.activeRequestsLock.Lock()
	delete(s.activeRequests, context.RequestID)
	s.activeRequestsLock.Unlock()

	log.Printf("send on channel")
	context.Done <- true
	log.Printf("done setting context to completed")
}

var acceptedResponse = &statusResponse{Status: "OK", HTTPStatusCode: 202}

func (s *ApiService) createRuntimeRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Route("/2018-06-01", func(r chi.Router) {
		r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("pong"))
		})

		r.Route("/runtime", func(r chi.Router) {
			r.
				// With(updateState("STATE_INIT_ERROR")).
				Post("/init/error", func(w http.ResponseWriter, r *http.Request) {
					debug("In /init/error")
					log.Fatalf("error handing unimplemented")
					// curContext = <-eventChan
					// handleErrorRequest(w, r)
					// curContext.EndInvoke(nil)
				})

			r.
				// With(updateState("STATE_INVOKE_NEXT")).
				Get("/invocation/next", func(w http.ResponseWriter, r *http.Request) {
					debug("In /invocation/next")

					closeNotify := w.(http.CloseNotifier).CloseNotify()
					go func() {
						<-closeNotify
						debug("Connection closed, sending ignore event")
						// eventChan <- &mockLambdaContext{Ignore: true}
					}()

					debug("Waiting for next event...")
					context := <-s.workChan
					// if context.Ignore {
					// 	debug("Ignore event received, returning")
					// 	w.Write([]byte{})
					// 	return
					// }

					context.LogStartRequest()

					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Lambda-Runtime-Aws-Request-Id", context.RequestID)
					w.Header().Set("Lambda-Runtime-Deadline-Ms", strconv.FormatInt(context.Deadline().UnixNano()/int64(time.Millisecond), 10))
					w.Header().Set("Lambda-Runtime-Invoked-Function-Arn", context.InvokedFunctionArn)
					w.Header().Set("Lambda-Runtime-Trace-Id", context.XAmznTraceID)

					if context.ClientContext != "" {
						w.Header().Set("Lambda-Runtime-Client-Context", context.ClientContext)
					}
					if context.CognitoIdentity != "" {
						w.Header().Set("Lambda-Runtime-Cognito-Identity", context.CognitoIdentity)
					}

					if context.LogType != "" {
						w.Header().Set("Docker-Lambda-Log-Type", context.LogType)
					}

					log.Printf("sending the data %v", context.EventBody)
					w.Write([]byte(context.EventBody))
				})

			r.Route("/invocation/{requestID}", func(r chi.Router) {
				// r.Use(awsRequestIDValidator)

				r.
					// With(updateState("STATE_INVOKE_RESPONSE")).
					Post("/response", func(w http.ResponseWriter, r *http.Request) {
						requestID := chi.URLParam(r, "requestID")
						log.Printf("have request id %s", requestID)

						context, contextFound := s.contextLookup(requestID)
						if !contextFound {
							render.Render(w, r, &errResponse{
								HTTPStatusCode: 500,
								ErrorType:      "Error", // Not sure what this would be in production?
								ErrorMessage:   "Invalid RequestID",
							})
							return
						}

						body, err := ioutil.ReadAll(r.Body)
						if err != nil {
							render.Render(w, r, &errResponse{
								HTTPStatusCode: 500,
								ErrorType:      "BodyReadError", // Not sure what this would be in production?
								ErrorMessage:   err.Error(),
							})
							return
						}
						r.Body.Close()

						debug("Setting Reply in /response")
						log.Printf("we have header %v", r.Header)
						log.Printf("we have context %v", r.Context())
						log.Printf("we have cookies %v", r.Cookies())

						context.Reply = &invokeResponse{Payload: body}

						// context.SetLogTail(r)
						context.SetInitEnd(r)
						s.contextCompleted(context)

						render.Render(w, r, acceptedResponse)
						w.(http.Flusher).Flush()
					})

				r.
					// With(updateState("STATE_INVOKE_ERROR")).
					Post("/error", handleErrorRequest)
			})
		})
	})
	return r
}

func handleErrorRequest(w http.ResponseWriter, r *http.Request) {
	/*
		lambdaErr := &lambdaError{}

		response := acceptedResponse

		body, err := ioutil.ReadAll(r.Body)
		if err != nil || json.Unmarshal(body, lambdaErr) != nil {
			debug("Could not parse error body as JSON")
			debug(body)
			response = &statusResponse{Status: "InvalidErrorShape", HTTPStatusCode: 299}
			lambdaErr = &lambdaError{Message: "InvalidErrorShape"}
		}
		r.Body.Close()

		errorType := r.Header.Get("Lambda-Runtime-Function-Error-Type")
		if errorType != "" {
			curContext.ErrorType = errorType
		}

		// TODO: Figure out whether we want to handle Lambda-Runtime-Function-XRay-Error-Cause

		debug("Setting Reply in handleErrorRequest")
		debug(lambdaErr)

		curContext.Reply = &invokeResponse{Error: lambdaErr}

		curContext.SetLogTail(r)
		curContext.SetInitEnd(r)

		render.Render(w, r, response)

		w.(http.Flusher).Flush()
	*/
}

func (s *ApiService) awsRequestIDValidator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := chi.URLParam(r, "requestID")

		// if requestID != curContext.RequestID {
		// 	render.Render(w, r, &errResponse{
		// 		HTTPStatusCode: 400,
		// 		ErrorType:      "InvalidRequestID",
		// 		ErrorMessage:   "Invalid request ID",
		// 	})
		// 	return
		// }

		ctx := context.WithValue(r.Context(), keyRequestID, requestID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type statusResponse struct {
	HTTPStatusCode int    `json:"-"`
	Status         string `json:"status"`
}

func (sr *statusResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, sr.HTTPStatusCode)
	return nil
}

type errResponse struct {
	HTTPStatusCode int    `json:"-"`
	ErrorType      string `json:"errorType,omitempty"`
	ErrorMessage   string `json:"errorMessage"`
}

func (e *errResponse) Render(w http.ResponseWriter, r *http.Request) error {
	render.Status(r, e.HTTPStatusCode)
	return nil
}

func fakeGUID() string {
	randBuf := make([]byte, 16)
	rand.Read(randBuf)

	hexBuf := make([]byte, hex.EncodedLen(len(randBuf))+4)

	hex.Encode(hexBuf[0:8], randBuf[0:4])
	hexBuf[8] = '-'
	hex.Encode(hexBuf[9:13], randBuf[4:6])
	hexBuf[13] = '-'
	hex.Encode(hexBuf[14:18], randBuf[6:8])
	hexBuf[18] = '-'
	hex.Encode(hexBuf[19:23], randBuf[8:10])
	hexBuf[23] = '-'
	hex.Encode(hexBuf[24:], randBuf[10:])

	hexBuf[14] = '1' // Make it look like a v1 guid

	return string(hexBuf)
}

func renderJSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}
	w.Write(buf.Bytes())
}

type requestContext struct {
	RequestID          string
	EventBody          string
	FnName             string
	Version            string // TODO how does this get populated?
	InvocationType     string
	ClientContext      string
	XAmznTraceID       string
	InvokedFunctionArn string
	Reply              *invokeResponse
	LogType            string
	LogTail            string // base64 encoded tail, no greater than 4096 bytes
	ErrorType          string // Unhandled vs Handled
	Done               chan bool
	CognitoIdentity    string
	Start              time.Time
	InvokeWait         time.Time
	InitEnd            time.Time
	TimeoutDuration    time.Duration
}

func (c *requestContext) SetInitEnd(r *http.Request) {
	invokeWaitHeader := r.Header.Get("Docker-Lambda-Invoke-Wait")
	if invokeWaitHeader != "" {
		invokeWaitMs, err := strconv.ParseInt(invokeWaitHeader, 10, 64)
		if err != nil {
			log.Fatal(fmt.Errorf("Could not parse Docker-Lambda-Invoke-Wait header as int. Error: %s", err))
			return
		}
		c.InvokeWait = time.Unix(0, invokeWaitMs*int64(time.Millisecond))
	}
	initEndHeader := r.Header.Get("Docker-Lambda-Init-End")
	if initEndHeader != "" {
		initEndMs, err := strconv.ParseInt(initEndHeader, 10, 64)
		if err != nil {
			log.Fatal(fmt.Errorf("Could not parse Docker-Lambda-Init-End header as int. Error: %s", err))
			return
		}
		c.InitEnd = time.Unix(0, initEndMs*int64(time.Millisecond))
	}
}

type invokeResponse struct {
	Payload []byte
	Error   *lambdaError
}

type lambdaError struct {
	Type       string       `json:"errorType,omitempty"`
	Message    string       `json:"errorMessage"`
	StackTrace []*string    `json:"stackTrace,omitempty"`
	Cause      *lambdaError `json:"cause,omitempty"`
}

func newRequestContext() *requestContext {
	return &requestContext{
		RequestID: fakeGUID(),
		Done:      make(chan bool),
	}
}

type key int

const (
	keyRequestID key = iota
)

func (c *requestContext) Deadline() time.Time {
	return c.Start.Add(c.TimeoutDuration)
}

func (c *requestContext) LogStartRequest() {
	c.InitEnd = time.Now()
	systemLog("START RequestId: " + c.RequestID + " Version: " + c.Version)
}

func (s *ApiService) addAPIRoutes(r *chi.Mux) *chi.Mux {
	r.Options("/*", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Origin") == "" {
			w.WriteHeader(403)
			return
		}
		w.Header().Set("x-amzn-requestid", fakeGUID())
		w.Header().Set("access-control-allow-origin", "*")
		w.Header().Set("access-control-expose-headers", "x-amzn-RequestId,x-amzn-ErrorType,x-amzn-ErrorMessage,Date,x-amz-log-result,x-amz-function-error")
		w.Header().Set("access-control-max-age", "172800")
		if r.Header.Get("Access-Control-Request-Headers") != "" {
			w.Header().Set("access-control-allow-headers", r.Header.Get("Access-Control-Request-Headers"))
		}
		if r.Header.Get("Access-Control-Request-Method") != "" {
			w.Header().Set("access-control-allow-methods", r.Header.Get("Access-Control-Request-Method"))
		}
		w.WriteHeader(200)
	})

	r.Post("/2015-03-31/functions/{function}/invocations", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("handing function invocation request")
		context := newRequestContext()

		if r.Header.Get("Origin") != "" {
			w.Header().Set("access-control-allow-origin", "*")
			w.Header().Set("access-control-expose-headers", "x-amzn-RequestId,x-amzn-ErrorType,x-amzn-ErrorMessage,Date,x-amz-log-result,x-amz-function-error")
		}

		if r.Header.Get("X-Amz-Invocation-Type") != "" {
			context.InvocationType = r.Header.Get("X-Amz-Invocation-Type")
		}
		if r.Header.Get("X-Amz-Client-Context") != "" {
			buf, err := base64.StdEncoding.DecodeString(r.Header.Get("X-Amz-Client-Context"))
			if err != nil {
				render.Render(w, r, &errResponse{
					HTTPStatusCode: 400,
					ErrorType:      "ClientContextDecodingError",
					ErrorMessage:   err.Error(),
				})
				return
			}
			context.ClientContext = string(buf)
		}
		if r.Header.Get("X-Amz-Log-Type") != "" {
			context.LogType = r.Header.Get("X-Amz-Log-Type")
		}

		if context.InvocationType == "DryRun" {
			w.Header().Set("x-amzn-RequestId", context.RequestID)
			w.Header().Set("x-amzn-Remapped-Content-Length", "0")
			w.WriteHeader(204)
			return
		}

		if body, err := ioutil.ReadAll(r.Body); err == nil {
			log.Printf("API received a request with body %s", body)
			context.EventBody = string(body)
		} else {
			render.Render(w, r, &errResponse{
				HTTPStatusCode: 500,
				ErrorType:      "BodyReadError",
				ErrorMessage:   err.Error(),
			})
			return
		}
		r.Body.Close()

		s.contextInvoke(context)

		if context.InvocationType == "Event" {
			w.Header().Set("x-amzn-RequestId", context.RequestID)
			w.Header().Set("x-amzn-Remapped-Content-Length", "0")
			w.Header().Set("X-Amzn-Trace-Id", context.XAmznTraceID)
			w.WriteHeader(202)
			go waitForContext(context)
			return
		}

		log.Printf("Waiting for context")
		waitForContext(context)
		log.Printf("Finished waiting for context")

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("x-amzn-RequestId", context.RequestID)
		w.Header().Set("x-amzn-Remapped-Content-Length", "0")
		w.Header().Set("X-Amz-Executed-Version", context.Version)
		w.Header().Set("X-Amzn-Trace-Id", context.XAmznTraceID)

		if context.LogType == "Tail" {
			// We assume context.LogTail is already base64 encoded
			w.Header().Set("X-Amz-Log-Result", context.LogTail)
		}

		if context.Reply.Error != nil {
			errorType := "Unhandled"
			if context.ErrorType != "" {
				errorType = context.ErrorType
			}
			w.Header().Set("X-Amz-Function-Error", errorType)
		}

		// Lambda will usually return the payload instead of an error if the payload exists
		if len(context.Reply.Payload) > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(int64(len(context.Reply.Payload)), 10))
			w.Write(context.Reply.Payload)
			return
		}

		if payload, err := json.Marshal(context.Reply.Error); err == nil {
			w.Header().Set("Content-Length", strconv.FormatInt(int64(len(payload)), 10))
			w.Write(payload)
		} else {
			render.Render(w, r, &errResponse{
				HTTPStatusCode: 500,
				ErrorType:      "ErrorMarshalError",
				ErrorMessage:   err.Error(),
			})
		}
	})
	return r
}

func waitForContext(context *requestContext) {
	<-context.Done
}
