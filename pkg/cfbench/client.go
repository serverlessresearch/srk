package cfbench

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/serverlessresearch/srk/pkg/srk"
	"github.com/sirupsen/logrus"
	"net/rpc"
)

type benchClient struct {
	log logrus.FieldLogger
}

func (bc *benchClient) RunBench(prov *srk.Provider, args *srk.BenchArgs) error {
	cert, err := srk.LoadKeyPair()
	//cert, err := tls.LoadX509KeyPair("certs/client.crt", "certs/client.key")
	//if err != nil {
	//	bc.log.Fatalf("client: loadkeys: %s", err)
	//}
	//if len(cert.Certificate) != 2 {
	//	bc.log.Fatal("client.crt should have 2 concatenated certificates: client + CA")
	//}
	ca, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return err
	}
	certPool := x509.NewCertPool()
	certPool.AddCert(ca)
	config := tls.Config{
		Certificates: []tls.Certificate{*cert},
		RootCAs:      certPool,
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", "54.202.88.203:6000", &config)
	if err != nil {
		bc.log.Fatalf("client: dial: %s", err)
	}
	defer conn.Close()
	bc.log.Println("client: connected to: ", conn.RemoteAddr())
	rpcClient := rpc.NewClient(conn)
	req := ConcurrencyScanRequest{
		FunctionName:     args.FName,
		BeginConcurrency: 2,
		EndConcurrency:   10,
		NumSteps:         5,
		StepDuration:     3,
	}

	var statusResp StatusResponse
	if err := rpcClient.Call("ExperimentRunner.Status", req, &statusResp); err != nil {
		return err
	}
	bc.log.Println("received status response %+v", statusResp)

	var resp ExperimentRunResponse
	if err := rpcClient.Call("ExperimentRunner.ConcurrencyScan", req, &resp); err != nil {
		return err
	}
	bc.log.Println("received scan response %+v", resp)

	return nil
}

func NewClient(logger srk.Logger) (*benchClient, error) {
	return &benchClient{logger}, nil
}