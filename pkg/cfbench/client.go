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
	conn, err := tls.Dial("tcp", "ec2-34-217-114-50.us-west-2.compute.amazonaws.com:6000", &config)
	//conn, err := tls.Dial("tcp", "localhost:6000", &config)
	//conn, err := tls.Dial("tcp", "34.219.59.68:6000", &config)
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
		NumLevels:        5,
		LevelDuration:    3,
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