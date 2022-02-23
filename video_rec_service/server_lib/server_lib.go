package server_lib

import (
	"context"

	umc "cs426.yale.edu/lab1/user_service/mock_client"
	pb "cs426.yale.edu/lab1/video_rec_service/proto"
	vmc "cs426.yale.edu/lab1/video_service/mock_client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	upb "cs426.yale.edu/lab1/user_service/proto"
	vpb "cs426.yale.edu/lab1/video_service/proto"
    ranker"cs426.yale.edu/lab1/ranker"

    // "cs426.yale.edu/lab1/video_rec_service/server"

    "log"
    "google.golang.org/grpc"
    // "flag"
    "sort"
    // "fmt"
    // "math" // Min function
    "sync"
    "time" // for latency
		"github.com/influxdata/tdigest"

    "google.golang.org/grpc/credentials/insecure"


)


// )

type VideoRecServiceOptions struct {
	// Server address for the UserService"
	UserServiceAddr string
	// Server address for the VideoService
	VideoServiceAddr string
	// Maximum size of batches sent to UserService and VideoService
	MaxBatchSize int
	// If set, disable fallback to cache
	DisableFallback bool
	// If set, disable all retries
	DisableRetry bool
}

func DefaultVideoRecServiceOptions() VideoRecServiceOptions {
	return VideoRecServiceOptions{
		UserServiceAddr:  "[::1]:8081",
		VideoServiceAddr: "[::1]:8082",
		MaxBatchSize:     250,
	}
}

type TrendingVideos struct {
	mu sync.RWMutex
	Vinfos []*vpb.VideoInfo
}




func (server *VideoRecServiceServer) getTrendingVideos() ([]*vpb.VideoInfo, uint64, error) {
	cxt, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var opts []grpc.DialOption
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// userServiceAddr is global
	conn, err := grpc.Dial(server.options.VideoServiceAddr, opts...)

	// handle errors
	if err != nil {
		// retry
		conn, err = grpc.Dial(server.options.VideoServiceAddr, opts...)
		if err != nil {
			log.Printf("Fail to dial: %v", err)
			return nil, 0, status.Errorf(codes.Unavailable, "Oops.. Dial Fail... in UserService")
		}
	}
	defer conn.Close()

	// create video client
	videoClient := vpb.NewVideoServiceClient(conn)

	response, err := videoClient.GetTrendingVideos(cxt, &vpb.GetTrendingVideosRequest{})
	if err != nil {
		log.Printf("Fail to dial: %v", err)
		return nil, 0, status.Errorf(codes.Unavailable, "Oops.. GetTrendingVideos fails.. ")
	}
	vids := response.Videos
	timeout := response.ExpirationTimeS

	batch_size := server.options.MaxBatchSize

	// get vinfos
	vinfos := []*vpb.VideoInfo{}
	k := 0
	for k < len(vids) {
		uplim := k + batch_size
		if uplim > len(vids) {
			uplim = len(vids)
		}
		res1, err1 := videoClient.GetVideo(context.Background(), &vpb.GetVideoRequest{VideoIds: vids[k:uplim]})

		if err1 != nil {
			// retry
			res1, err1 = videoClient.GetVideo(context.Background(), &vpb.GetVideoRequest{VideoIds: vids[k:uplim]})
			if err1 != nil {
				log.Printf("Fail to GetTendingVideo: %v", err1)
				return nil, 0, status.Errorf(codes.Unavailable, "Oops.. GetTrendingVideo() Failed!!")
			}

		}
		vinfos = append(vinfos, res1.Videos...)
		k += batch_size
	}

	return vinfos, timeout, nil

}


func (server *VideoRecServiceServer) getTrendingVideosMock() ([]*vpb.VideoInfo, uint64, error) {
	cxt, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	videoClient := server.Mock_vclient

	response, err := videoClient.GetTrendingVideos(cxt, &vpb.GetTrendingVideosRequest{})
	if err != nil {
		log.Printf("Fail to get trending videos with mock: %v", err)
		return nil, 0, status.Errorf(codes.Unavailable, "Oops.. GetTrendingVideosMock fails.. %v", err)
	}
	vids := response.Videos
	timeout := response.ExpirationTimeS

	batch_size := server.options.MaxBatchSize

	// get vinfos
	vinfos := []*vpb.VideoInfo{}
	k := 0
	for k < len(vids) {
		uplim := k + batch_size
		if uplim > len(vids) {
			uplim = len(vids)
		}
		res1, err1 := videoClient.GetVideo(context.Background(), &vpb.GetVideoRequest{VideoIds: vids[k:uplim]})

		if err1 != nil {
			// retry
			res1, err1 = videoClient.GetVideo(context.Background(), &vpb.GetVideoRequest{VideoIds: vids[k:uplim]})
			if err1 != nil {
				log.Printf("Fail to GetTendingVideo: %v", err1)
				return nil, 0, status.Errorf(codes.Unavailable, "Oops.. GetTrendingVideo() Failed!!")
			}

		}
		vinfos = append(vinfos, res1.Videos...)
		k += batch_size
	}
	log.Println("Mock Get Trending Videos SUccess!!")
	return vinfos, timeout, nil

}





func (server *VideoRecServiceServer) UpdateTrendingVideos() {

	timeout := uint64(0)

	for {
		vinfos, t, err := server.getTrendingVideos()
		if err != nil {
			log.Printf("Fail to GetTendingVideo: %v, Backing off 10 secs...", err)
			time.Sleep(10 * time.Second) // push back 10 seconds
			continue
		}
		server.Trending_videos.mu.Lock()
		// log.Printf("Updated Cache for Trending Videos, timeout is %v",t)
		// log.Printf("The time now is %v",uint64(time.Now().Unix()))
		timeout = t
		server.Trending_videos.Vinfos = vinfos
		server.Trending_videos.mu.Unlock()
		time.Sleep(time.Duration(timeout - uint64(time.Now().Unix())) * time.Second)
	}
}


func (server *VideoRecServiceServer) UpdateTrendingVideosMock() {

	timeout := uint64(0)

	for {
		vinfos, t, err := server.getTrendingVideosMock()
		if err != nil {
			log.Printf("Fail to GetTendingVideoMock: %v, Backing off 10 secs...", err)
			time.Sleep(10 * time.Second) // push back 10 seconds
			continue
		}
		server.Trending_videos.mu.Lock()
		// log.Printf("Updated Cache for Trending Videos, timeout is %v",t)
		// log.Printf("The time now is %v",uint64(time.Now().Unix()))
		timeout = t
		server.Trending_videos.Vinfos = vinfos
		server.Trending_videos.mu.Unlock()
		time.Sleep(time.Duration(timeout - uint64(time.Now().Unix())) * time.Second)
	}
}


type VideoRecServiceServer struct {
	pb.UnimplementedVideoRecServiceServer
	options VideoRecServiceOptions
	// Add any data you want here
	mu sync.RWMutex
	Total_requests_ uint64
	Total_error_ uint64
	Total_active_ uint64
	UserService_error_ uint64
	VideoService_error_ uint64
	Average_latency float32 //milliseconds
	Stale_response_ uint64
	Trending_videos TrendingVideos
	Mock_uclient *umc.MockUserServiceClient
	Mock_vclient *vmc.MockVideoServiceClient
	td  *tdigest.TDigest
}


func MakeVideoRecServiceServer(options VideoRecServiceOptions) (*VideoRecServiceServer) {
	s := &VideoRecServiceServer{
		options: options,
		// Add any data to initialize here
		Total_requests_: 0,
		Total_error_: 0,
		Total_active_: 0,
		UserService_error_: 0,
		VideoService_error_: 0,
		Average_latency: 0.0,
		Stale_response_:0,
		// Trending Videos
		Trending_videos: TrendingVideos{ Vinfos: []*vpb.VideoInfo{} },
		Mock_uclient: nil,
		Mock_vclient: nil,
		td: tdigest.NewWithCompression(100),
	}
	go s.UpdateTrendingVideos()
	return s
}

func (server *VideoRecServiceServer)QueryTrendingVideos() ([]*vpb.VideoInfo) {
	server.Trending_videos.mu.RLock()
	defer server.Trending_videos.mu.RUnlock()
	return server.Trending_videos.Vinfos
}

func MakeVideoRecServiceServerWithMocks(
	options VideoRecServiceOptions,
	mockUserServiceClient *umc.MockUserServiceClient,
	mockVideoServiceClient *vmc.MockVideoServiceClient,
) *VideoRecServiceServer {
	// Implement your own logic here

	s := &VideoRecServiceServer{
		options: options,
		// Add any data to initialize here
		Total_requests_: 0,
		Total_error_: 0,
		Total_active_: 0,
		UserService_error_: 0,
		VideoService_error_: 0,
		Average_latency: 0.0,
		Stale_response_:0,
		// Trending Videos
		Trending_videos: TrendingVideos{ Vinfos: []*vpb.VideoInfo{} },
		Mock_uclient: mockUserServiceClient,
		Mock_vclient: mockVideoServiceClient,
		td: tdigest.NewWithCompression(100),
	}
	go s.UpdateTrendingVideosMock()
	return s
}



// define UInts for sort
type Uints []uint64

func (u Uints) Len() int{
	return len(u)
}

func (u Uints) Less (i, j int) bool {
	return u[i] < u[j]
}
func (u Uints) Swap(i, j int) {
	u[i], u[j] = u[j], u[i]
}

func (server *VideoRecServiceServer)ExitWithError (server_type string) {
	server.mu.Lock()
	server.Total_error_ += 1
	server.Total_active_ -= 1
	if server_type == "user" {
		server.UserService_error_ += 1
	}
	if server_type == "video" {
		server.VideoService_error_ += 1
	}
	defer server.mu.Unlock()
	// log.Printf("VideoRecServer exited with error... What Happens?")
	return
}


// return the proper type of error using error code
// and the fallback
func (server *VideoRecServiceServer)error_fallback(limit int32) (*pb.GetTopVideosResponse) {
	vinfos := server.QueryTrendingVideos()
	if len(vinfos) == 0{  // no cache yet
		return nil
	}
	kk := limit
	if kk == 0 {
		kk = int32(len(vinfos))
	}
	// update fallback calls
	server.mu.Lock()
	server.Stale_response_ += 1
	server.Total_error_ -= 1   // decrease the total error as we have stale response
	server.mu.Unlock()
	return &pb.GetTopVideosResponse{Videos:vinfos[:kk], StaleResponse: true}
}


func (server *VideoRecServiceServer) GetTopVideos_Mock(req *pb.GetTopVideosRequest, uclient *umc.MockUserServiceClient,vclient *vmc.MockVideoServiceClient) (*pb.GetTopVideosResponse, error) {
	server.mu.Lock()
	server.Total_requests_ += 1
	server.Total_active_ += 1
	server.mu.Unlock()

	start := time.Now().UnixNano()
	userClient := uclient
	videoClient := vclient


	response, err := userClient.GetUser(context.Background(), &upb.GetUserRequest {UserIds: []uint64{req.UserId}})
	if err != nil {
		// retry
		response, err = userClient.GetUser(context.Background(), &upb.GetUserRequest {UserIds: []uint64{req.UserId}})
		if err != nil{
			log.Printf("Fail to get user: %v", err)
			server.ExitWithError("user")
			return server.error_fallback(req.Limit), status.Errorf(codes.Unavailable, "GetUser() Fail in GetTopVideos()...")
		}
	}

	subscribed_ids := response.Users[0].SubscribedTo
	orig_uinfo := response.Users[0]


	// From here: need to check MaxBatchSize and do batching
	batch_size := server.options.MaxBatchSize
	uinfos := []*upb.UserInfo{}
	j := 0
	for j < len(subscribed_ids) {
		uplim := j+batch_size
		if len(subscribed_ids) < uplim {
			uplim = len(subscribed_ids)
		}
		res, err3 := userClient.GetUser(context.Background(), &upb.GetUserRequest{UserIds: subscribed_ids[j:uplim]})
		if err3 != nil {
			// retry
			res, err3 = userClient.GetUser(context.Background(), &upb.GetUserRequest{UserIds: subscribed_ids[j:uplim]})
			if err3 != nil {
				log.Printf("Fail to get user: %v", err3)
				server.ExitWithError("user")
				return server.error_fallback(req.Limit), status.Errorf(codes.Unavailable, "GetUser() Fail querying users")
			}

		}
		uinfos = append(uinfos, res.Users...)
		j += batch_size
	}


	vids := []uint64{}
	// visited used to record, removing dupicate video ids
	visited := make(map[uint64]bool)
	for _, uinfo := range uinfos {
		for _, vid := range uinfo.LikedVideos {
			if _, value := visited[vid]; !value {
				visited[vid] = true
				vids = append(vids, vid)
			}
		}
	}

	// Querying the VideoInfo by batches
	// Recall that batch_size is maximum batch size
	vinfos := []*vpb.VideoInfo{}
	k := 0
	for k < len(vids) {
		uplim := k + batch_size
		if uplim > len(vids) {
			uplim = len(vids)
		}
		res1, err1 := videoClient.GetVideo(context.Background(), &vpb.GetVideoRequest{VideoIds: vids[k:uplim]})

		if err1 != nil {
			// retry
			res1, err1 = videoClient.GetVideo(context.Background(), &vpb.GetVideoRequest{VideoIds: vids[k:uplim]})
			if err1 != nil {
				log.Printf("Fail to GetVideo: %v", err1)
				server.ExitWithError("video")
				return server.error_fallback(req.Limit), status.Errorf(codes.Unavailable, "Oops.. GetVideo() Failed!!")
			}

		}
		vinfos = append(vinfos, res1.Videos...)
		k += batch_size
	}


  	// Reminder:
  	// Original User Id is : req.UserId
	m := make(map[uint64](*vpb.VideoInfo))

	orig_coeff := orig_uinfo.UserCoefficients
	for _, vinfo := range vinfos {
		rkr := ranker.BcryptRanker{}
		score := rkr.Rank(orig_coeff, vinfo.VideoCoefficients)
		m[score] = vinfo
	}

	var a []uint64
	for k := range m {
		a = append(a, k)
	}

	sort.Sort(sort.Reverse(Uints(a)))

	kk := req.Limit
	sorted_vinfos := make([]*vpb.VideoInfo, kk)
	for idx, k := range a[:kk] {
		sorted_vinfos[idx] = m[k]
	}

	if kk == 0 {
		kk = int32(len(sorted_vinfos))
	}

	t := time.Now().UnixNano()
	elapsed := float32((t - start)/ 1000000)

	server.mu.Lock()
	server.Total_active_ -= 1
	server.td.Add(float64(elapsed), 1)
	server.Average_latency = (server.Average_latency * float32(server.Total_requests_ - server.Total_error_ - server.Total_active_ - 1) + elapsed) / float32(server.Total_requests_ - server.Total_active_ - server.Total_error_)
	server.mu.Unlock()

	return &pb.GetTopVideosResponse{Videos:sorted_vinfos[:kk]}, nil

}


func (server *VideoRecServiceServer) GetTopVideos(ctx context.Context, req *pb.GetTopVideosRequest,
) (*pb.GetTopVideosResponse, error) {

	// check mock or not
	// If mock then flag is true
	mock_flag := server.Mock_uclient != nil
	if mock_flag {
		// log.Printf("Using Mock !")
		result, err := server.GetTopVideos_Mock(req, server.Mock_uclient, server.Mock_vclient)
		if err != nil{
			log.Printf("Something failed in Mock GetTopVideos: %v", err)
		}

		return result, err
	}


	server.mu.Lock()
	server.Total_requests_ += 1
	server.Total_active_ += 1
	server.mu.Unlock()

	start := time.Now().UnixNano()

	opts := []grpc.DialOption{}
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// Create a userClient object
	conn, err := grpc.Dial(server.options.UserServiceAddr, opts...)

	// handle errors
	if err != nil {
		// retry
		conn, err = grpc.Dial(server.options.UserServiceAddr, opts...)
		if err != nil {
			log.Printf("Fail to dial: %v", err)
			server.ExitWithError("user")
			return server.error_fallback(req.Limit), status.Errorf(codes.Unavailable, "Oops.. Dial Fail... in UserService %v", err)
		}
	}
	defer conn.Close()
	userClient := upb.NewUserServiceClient(conn)


	response, err := userClient.GetUser(context.Background(), &upb.GetUserRequest {UserIds: []uint64{req.UserId}})
	if err != nil {
		// retry
		response, err = userClient.GetUser(context.Background(), &upb.GetUserRequest {UserIds: []uint64{req.UserId}})
		if err != nil{
			log.Printf("Fail to get user: %v", err)
			server.ExitWithError("user")
			return server.error_fallback(req.Limit), status.Errorf(codes.Unavailable, "GetUser() Fail in GetTopVideos()... %v", err)
		}
	}

	subscribed_ids := response.Users[0].SubscribedTo
	orig_uinfo := response.Users[0]



	// From here: need to check MaxBatchSize and do batching
	batch_size := server.options.MaxBatchSize
	uinfos := []*upb.UserInfo{}
	j := 0
	for j < len(subscribed_ids) {
		uplim := j+batch_size
		if len(subscribed_ids) < uplim {
			uplim = len(subscribed_ids)
		}
		res, err3 := userClient.GetUser(context.Background(), &upb.GetUserRequest{UserIds: subscribed_ids[j:uplim]})
		if err3 != nil {
			// retry
			res, err3 = userClient.GetUser(context.Background(), &upb.GetUserRequest{UserIds: subscribed_ids[j:uplim]})
			if err3 != nil {
				log.Printf("Fail to get user: %v", err3)
				server.ExitWithError("user")
				return server.error_fallback(req.Limit), status.Errorf(codes.Unavailable, "GetUser() Fail querying users")
			}

		}
		uinfos = append(uinfos, res.Users...)
		j += batch_size
	}


	vids := []uint64{}
	// visited used to record, removing dupicate video ids
	visited := make(map[uint64]bool)
	for _, uinfo := range uinfos {
		for _, vid := range uinfo.LikedVideos {
			if _, value := visited[vid]; !value {
				visited[vid] = true
				vids = append(vids, vid)
			}
		}
	}


	// Connect to Video Service
	// videoClient := server.Mock_vclient
	// if !mock_flag{
	conn2, err2 := grpc.Dial(server.options.VideoServiceAddr, opts...)
	if err2 != nil {
		// retry
		conn2, err2 = grpc.Dial(server.options.VideoServiceAddr, opts...)
		if err2 != nil {
			log.Printf("Fail to dial - video service: %v", err2)
			server.ExitWithError("video")
			return server.error_fallback(req.Limit), status.Errorf(codes.Unavailable, "Oops.. Dial Fail...VideoService")
		}
	}
	defer conn2.Close()
	videoClient := vpb.NewVideoServiceClient(conn2)
	// }



	// Querying the VideoInfo by batches
	// Recall that batch_size is maximum batch size
	vinfos := []*vpb.VideoInfo{}
	k := 0
	for k < len(vids) {
		uplim := k + batch_size
		if uplim > len(vids) {
			uplim = len(vids)
		}
		res1, err1 := videoClient.GetVideo(context.Background(), &vpb.GetVideoRequest{VideoIds: vids[k:uplim]})

		if err1 != nil {
			// retry
			res1, err1 = videoClient.GetVideo(context.Background(), &vpb.GetVideoRequest{VideoIds: vids[k:uplim]})
			if err1 != nil {
				log.Printf("Fail to GetVideo: %v", err1)
				server.ExitWithError("video")
				return server.error_fallback(req.Limit), status.Errorf(codes.Unavailable, "Oops.. GetVideo() Failed!!")
			}

		}
		vinfos = append(vinfos, res1.Videos...)
		k += batch_size
	}

  	// Reminder:
  	// Original User Id is : req.UserId
  	m := make(map[uint64](*vpb.VideoInfo))

  	orig_coeff := orig_uinfo.UserCoefficients
  	for _, vinfo := range vinfos {
  		rkr := ranker.BcryptRanker{}
  		score := rkr.Rank(orig_coeff, vinfo.VideoCoefficients)
  		m[score] = vinfo
  	}

  	var a []uint64
  	for k := range m {
  		a = append(a, k)
  	}

  	sort.Sort(sort.Reverse(Uints(a)))

 	  kk := req.Limit
  	sorted_vinfos := make([]*vpb.VideoInfo, kk)
  	for idx, k := range a[:kk] {
  		sorted_vinfos[idx] = m[k]
  	}

		if kk == 0 {
			kk = int32(len(sorted_vinfos))
		}

  	t := time.Now().UnixNano()
  	elapsed := float32((t - start)/ 1000000)


  	server.mu.Lock()
  	server.Total_active_ -= 1
		server.td.Add(float64(elapsed), 1)
  	server.Average_latency = (server.Average_latency * float32(server.Total_requests_ - server.Total_error_ - server.Total_active_ - 1) + elapsed) / float32(server.Total_requests_ - server.Total_active_ - server.Total_error_)
  	server.mu.Unlock()

  	return &pb.GetTopVideosResponse{Videos:sorted_vinfos[:kk]}, nil


}



// TODO: Implement the GetStats functions
func (server *VideoRecServiceServer) GetStats (ctx context.Context,
	req *pb.GetStatsRequest,)(*pb.GetStatsResponse, error) {

	server.mu.RLock()
	defer server.mu.RUnlock()
	res := &pb.GetStatsResponse {
		TotalRequests: server.Total_requests_,
		TotalErrors: server.Total_error_,
		ActiveRequests: server.Total_active_,
		UserServiceErrors: server.UserService_error_,
		VideoServiceErrors: server.VideoService_error_,
		AverageLatencyMs: server.Average_latency,
		StaleResponses: server.Stale_response_,
		P99LatencyMs: float32(server.td.Quantile(0.99)),
	}

	return res, nil

}
