package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/iambpn/chirpc/v1"
)

const addr = ":8888"

type MessageResponse struct {
	Message string `json:"message"`
}

func streamDoggoImage(_ *http.Request) (*chirpc.StreamResponse, *chirpc.ErrorResponse) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return nil, &chirpc.ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Errors:     []string{"failed to resolve file path for stream example"},
		}
	}

	doggoPath := filepath.Join(filepath.Dir(thisFile), "doggo.jpg")
	file, err := os.Open(doggoPath)
	if err != nil {
		return nil, &chirpc.ErrorResponse{
			StatusCode: http.StatusInternalServerError,
			Errors:     []string{"failed to open doggo.jpg: " + err.Error()},
		}
	}

	stream := chirpc.ReaderToStream(file, file.Close)

	return &chirpc.StreamResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Content-Type":  "image/jpeg",
			"X-Stream-File": "doggo.jpg",
		},
		Stream: stream,
	}, nil
}

func main() {
	rpcRouter := chirpc.NewRPCRouter()
	chirpc.AddStreamHandler[[]byte](rpcRouter, chirpc.MethodGet, "/stream/doggo.jpg", streamDoggoImage)

	err := chirpc.GenerateRPCSchema(rpcRouter)

	if err != nil {
		println("Error generating types:", err.Error())
		return
	}

	fmt.Println("RPC schema generated successfully")

	server := rpcRouter.GetHttpServer()
	server.Addr = addr

	println("Starting server on", addr)
	if err := server.ListenAndServe(); err != nil {
		panic(err)
	}
}
