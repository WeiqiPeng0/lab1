
package main

import (
	"context"
	// "flag"
	// "hash/fnv"
	// "log"
	"time"
	// "fmt"
	//
	// upb "cs426.yale.edu/lab1/user_service/proto"
	pb "cs426.yale.edu/lab1/video_rec_service/proto"
	// vpb "cs426.yale.edu/lab1/video_service/proto"
	// "google.golang.org/grpc"
	// "google.golang.org/grpc/credentials/insecure"
	"testing"
	"reflect"

	vmc "cs426.yale.edu/lab1/video_service/mock_client"
	umc "cs426.yale.edu/lab1/user_service/mock_client"
	vsl "cs426.yale.edu/lab1/video_service/server_lib"
	usl "cs426.yale.edu/lab1/user_service/server_lib"
	vrsl "cs426.yale.edu/lab1/video_rec_service/server_lib"


)

// Create Connection with Mock
func GetMockServer(batch_size int) (*vrsl.VideoRecServiceServer){
	voptions := vsl.DefaultVideoServiceOptions()
	mock_vclient := vmc.MakeMockVideoServiceClient(*voptions)
	uoptions := usl.DefaultUserServiceOptions()
	mock_uclient := umc. MakeMockUserServiceClient(*uoptions)
	options := vrsl.VideoRecServiceOptions{
		UserServiceAddr:  "[::1]:8081",
		VideoServiceAddr: "[::1]:8082",
		MaxBatchSize:     batch_size,
	}
	vr_server := vrsl.MakeVideoRecServiceServerWithMocks(options, mock_uclient, mock_vclient)
	return vr_server
}


func TestMock(t *testing.T) {
	// log.Printf("Testing Mock Server..")
	voptions := vsl.DefaultVideoServiceOptions()
	mock_vclient := vmc.MakeMockVideoServiceClient(*voptions)
	uoptions := usl.DefaultUserServiceOptions()
	mock_uclient := umc. MakeMockUserServiceClient(*uoptions)
	options := vrsl.DefaultVideoRecServiceOptions()
	vr_server := vrsl.MakeVideoRecServiceServerWithMocks(options, mock_uclient, mock_vclient)

	var temp vrsl.VideoRecServiceServer
	if reflect.TypeOf(*vr_server) != reflect.TypeOf(temp) {
		t.Fatalf("GetMockServer Failed!!!")
	}
}

// Basic functions of get top videos
// The test cases are taken from frontend
func TestGetTopVideos2(t *testing.T) {
	limit := int32(8)
	userId := uint64(203584)
	server := GetMockServer(250)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	out, err := server.GetTopVideos(
		ctx,
		&pb.GetTopVideosRequest{UserId: userId, Limit: limit},
	)
	defer cancel()
	if err != nil {
		t.Fatalf(
			"Error retrieving video recommendations for user (userId %d): %v\n",
			userId,
			err,
		)
	}
	videos := out.Videos

	if len(videos) != int(limit) || videos[0].VideoId != 1015 || videos[1].Author != "Belle Harber" ||
		videos[2].VideoId != 1268 || videos[3].Url != "https://video-data.localhost/blob/1040" || videos[4].Title != "charming towards" {
		t.Fatalf("Test case 2 with limit %d FAILED! Wrong recommendations", int(limit))
	}

}



func TestGetTopVideos1(t *testing.T) {
	limit := int32(5)
	userId := uint64(204054)
	server := GetMockServer(250)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	out, err := server.GetTopVideos(
		ctx,
		&pb.GetTopVideosRequest{UserId: userId, Limit: limit},
	)
	defer cancel()
	if err != nil {
		t.Fatalf(
			"Error retrieving video recommendations for user (userId %d): %v\n",
			userId,
			err,
		)
	}
	videos := out.Videos

	if len(videos) != int(limit)|| videos[0].VideoId != 1085 || videos[1].Author != "Renee Blick" ||
		videos[2].VideoId != 1106 || videos[3].Url != "https://video-data.localhost/blob/1211" || videos[4].Title != "evil elsewhere" {
		t.Fatalf("Test case 1 FAILED! Wrong recommendations")
	}

}

// Decreate the max batch size for testing
func TestBatch(t *testing.T) {
	limit := int32(8)
	userId := uint64(204054)
	server := GetMockServer(5)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	out, err := server.GetTopVideos(
		ctx,
		&pb.GetTopVideosRequest{UserId: userId, Limit: limit},
	)
	defer cancel()
	if err != nil {
		t.Fatalf(
			"Error retrieving video recommendations for user (userId %d): %v\n",
			userId,
			err,
		)
	}
	videos := out.Videos

	if len(videos) != int(limit)|| videos[0].VideoId != 1085 || videos[1].Author != "Renee Blick" ||
		videos[2].VideoId != 1106 || videos[3].Url != "https://video-data.localhost/blob/1211" || videos[4].Title != "evil elsewhere" {
		t.Fatalf("Test case 1 FAILED! Wrong recommendations")
	}

}

// Check getStats
func TestGetStats(t *testing.T) {
	userId := uint64(204054)
	server := GetMockServer(50)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	for i:=0; i< 10; i+=1 {
		_ , err := server.GetTopVideos(
			ctx,
			&pb.GetTopVideosRequest{UserId: userId, Limit: 5},
		)
		defer cancel()
		if err != nil {
			t.Fatalf(
				"Error retrieving video recommendations for user (userId %d): %v\n",
				userId,
				err,
			)
		}
	}
	stats, _ := server.GetStats(ctx, &pb.GetStatsRequest{})
	if stats.TotalRequests != 10 || stats.TotalErrors != 0 {
			t.Fatalf("GetStats FAILED!")
	}

}

// Create a bad server to simulate when the user client is down
// It seems that the options are gloabl
// so if I set one failure rate 1, both would fail
// For test, just imagine both shall fail...
func GetMockServerBad(batch_size int) (*vrsl.VideoRecServiceServer){
	voptions := vsl.VideoServiceOptions{
		Seed:                 42,
		TtlSeconds:           60,
		SleepNs:              0,
		FailureRate:          1,
		ResponseOmissionRate: 0,
		MaxBatchSize:         250,
	}
	mock_vclient := vmc.MakeMockVideoServiceClient(voptions) //vmc.MakeMockVideoServiceClient(*voptions)
	uoptions := usl.UserServiceOptions{
		Seed:                 42,
		SleepNs:              0,
		FailureRate:          1, //always return false
		ResponseOmissionRate: 0,
		MaxBatchSize:         250,
	}
	mock_uclient := umc.MakeMockUserServiceClient(uoptions)
	options := vrsl.VideoRecServiceOptions{
		UserServiceAddr:  "[::1]:8081",
		VideoServiceAddr: "[::1]:8082",
		MaxBatchSize:     batch_size,
	}
	vr_server := vrsl.MakeVideoRecServiceServerWithMocks(options, mock_uclient, mock_vclient)
	return vr_server
}

// Test the fallback mechanism when user service is down
func TestFallBack(t *testing.T) {
	server := GetMockServerBad(50)
	userId := uint64(204054)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	out, _ := server.GetTopVideos(
		ctx,
		&pb.GetTopVideosRequest{UserId: userId, Limit: 5},
	)
	defer cancel()

	if out == nil {
		// log.Println("Fallback is not triggered!!")
		stats, _ := server.GetStats(ctx, &pb.GetStatsRequest{})
		if stats.TotalRequests != 1 || stats.TotalErrors != 1 || stats.UserServiceErrors != 1  {
				t.Fatalf("GetStats FAILED in testing fall back! In the condition that fallback not triggered!")
		}
	} else {

		  t.Fatalf("Fallback should not have been triggered... both services shut down")
	}

}

func TestFallBackError(t *testing.T) {
	server := GetMockServerBad(60)
	userId := uint64(204054)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	out, err := server.GetTopVideos(
		ctx,
		&pb.GetTopVideosRequest{UserId: userId, Limit: 5},
	)
	defer cancel()

	if out == nil {
		if err == nil  {
				t.Fatalf("Error handle FAILED in testing fall back! In the condition that fallback not triggered!")
		}
	} else {
			t.Fatalf("Fallback should not have been triggered... both services shut down")
	}

}
