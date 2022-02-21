package main

import (
	// "context"
	"fmt"
	// umc "cs426.yale.edu/lab1/user_service/mock_client"
	// pb "cs426.yale.edu/lab1/video_rec_service/proto"
	// vmc "cs426.yale.edu/lab1/video_service/mock_client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	upb "cs426.yale.edu/lab1/user_service/proto"
	// vpb "cs426.yale.edu/lab1/video_service/proto"
  	// "cs426.yale.edu/lab1/ranker"
	
	
)

func main() {
	fmt.Println("Executing the main function.... \n")


	var opts []grpc.DialOption
	// userServiceAddr is global
	conn, err := grpc.Dial(*userServiceAddr, opts...)

	// handle errors
	if err != nil {
		log.Printf("Fail to dial: %v", err)
		return nil, status.Errorf(codes.Unavailable, "Oops.. Dail Fail...")
	}
	
	defer conn.Close()

	userClient := upb.NewUserServiceClient(conn)

}