package main

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	maxWorkers  = 10
	jobQueue    = make(chan Job, 1000)
	resultChan  = make(chan DetectionResults, 1000)
	redisClient *redis.Client
	ctx         = context.Background()
	wg          sync.WaitGroup
	resultsFile *os.File
	fileMutex   sync.Mutex
)

type Job struct {
	ImageName  string
	ImageBytes []byte
}

type Results struct {
	TotalImages int
	Processed   int
	Cached      int
	Faces       []DetectionResults
}

type DetectionResults map[string]interface{}

type FinalResp struct {
	JobId  string
	Result Results
}

var face_detection_Results Results

func main() {
	// Initialize Redis client
	redisHost := os.Getenv("REDIS_HOST")
	redisPort := os.Getenv("REDIS_PORT")
	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: "", // No password set
		DB:       0,  // Use default DB
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize results file
	var err error
	resultsFile, err = os.OpenFile("results.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		log.Fatalf("Failed to open the results file: %v", err)
	}
	defer resultsFile.Close()

	// Start worker goroutines
	startWorkers(maxWorkers)

	// Initialize Gin router
	r := gin.Default()
	port := "8080"
	r.POST("/detect", handleDetect)
	r.POST("/clear-cache", handleClearCache)
	r.GET("/find-job/:jobId", getResultsById)
	r.Run(":" + port)
}

func startWorkers(maxWorkers int) {
	for i := 0; i < maxWorkers; i++ {
		go func(workerId int) {
			for job := range jobQueue {
				result := processJob(job)
				resultChan <- result
			}
		}(i)
	}
}

func handleDetect(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		handleError(c, http.StatusBadRequest, "Failed to parse multipart form")
		fmt.Println("Error is", err)
		return
	}

	files := form.File["image_file"]
	if len(files) == 0 {
		handleError(c, http.StatusBadRequest, "No files uploaded")
		return
	}

	wg.Add(len(files))

	go func() {
		defer wg.Wait()
		for result := range resultChan {
			writeResultsToFile(result)
			wg.Done()
		}
	}()

	for _, fh := range files {
		f, err := fh.Open()
		if err != nil {
			handleError(c, http.StatusInternalServerError, "Unable to open the image file: "+fh.Filename)
			wg.Done()
			continue
		}

		defer f.Close()

		// Read image data
		buf := bytes.NewBuffer(nil)
		if _, err := buf.ReadFrom(f); err != nil {
			handleError(c, http.StatusInternalServerError, "Failed to read image file: "+fh.Filename)
			wg.Done()
			f.Close()
			continue
		}

		imageBytes := buf.Bytes()

		hash := md5.New()

		hash.Write(imageBytes)

		cacheKey := fmt.Sprintf("imagekey:%s", hex.EncodeToString(hash.Sum(nil)))
		cachedResult, err := redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			var result DetectionResults
			if err := json.Unmarshal([]byte(cachedResult), &result); err != nil {
				log.Printf("Failed to unmarshal cached result for %s: %v", fh.Filename, err)
				handleError(c, http.StatusInternalServerError, "Failed to retrieve cached result for the file: "+fh.Filename)
				wg.Done()
				continue
			}

			result["ImageName"] = fh.Filename
			result["ResultType"] = "Cached"
			writeResultsToFile(result) // Return cached result
			wg.Done()
			continue
		} else if err != redis.Nil {
			log.Printf("Failed to retrieve cached result for %s from Redis: %v", fh.Filename, err)
			handleError(c, http.StatusInternalServerError, "Failed to retrieve cached result for the file: "+fh.Filename)
			wg.Done()
			continue
		}
		job := Job{ImageBytes: imageBytes, ImageName: fh.Filename}
		jobQueue <- job
	}
	wg.Wait()
	id := uuid.New()
	jsonResp := FinalResp{JobId: id.String(), Result: face_detection_Results}

	data, err := json.Marshal(&jsonResp)
	if err != nil {
		panic(err)
	}
	fmt.Println(string(data))

	err = redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: "FaceDetectionStream",
		Values: map[string]interface{}{"JobId": id.String(), "Result": string(data)},
	}).Err()

	if err != nil {
		fmt.Println("failed to add data to stream, ", err)
	}
	b := new(bytes.Buffer)

	_ = json.NewEncoder(b).Encode(jsonResp)

	c.Data(http.StatusOK, "application/json", b.Bytes())

	face_detection_Results = Results{}
}

func handleClearCache(c *gin.Context) {
	err := redisClient.FlushAll(ctx).Err()
	if err != nil {
		log.Printf("Failed to flush Redis database: %v", err)
		handleError(c, http.StatusInternalServerError, "Failed to clear the Redis Cache")
		return
	}
	log.Printf("Redis cache flushed successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Redis cache cleared successfully"})
}

func getResultsById(c *gin.Context) {
	id := c.Param("jobId")

	fmt.Println("Job id is, ", id)

	streams, err := redisClient.XRead(ctx, &redis.XReadArgs{
		Streams: []string{"FaceDetectionStream", ">"},
	}).Result()

	if err != nil {
		panic(err)
	}

	for _, stream := range streams {
		fmt.Println("Stream:", stream.Stream)
		for _, message := range stream.Messages {
			fmt.Println("ID:", message.ID)
			for k, v := range message.Values {
				fmt.Println(k, v)
			}
		}
	}
}

func processJob(job Job) DetectionResults {
	// Call Python script for face detection using MediaPipe
	cmd := exec.Command("python", "face_detection_mediapipe.py", job.ImageName)
	cmd.Stdin = bytes.NewReader(job.ImageBytes)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		log.Printf("Failed to run face detection: %v", err)
		return DetectionResults{}
	}

	// Parse Python script output
	var result DetectionResults
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		log.Printf("Failed to parse face detection results: %v", err)
		return DetectionResults{}
	}

	hash := md5.New()

	hash.Write(job.ImageBytes)

	// Cache result in Redis for 1 hour (adjust TTL as needed)
	//cacheKey := fmt.Sprintf("image:%s", job.ImageName)
	cacheKey := fmt.Sprintf("imagekey:%s", hex.EncodeToString(hash.Sum(nil)))
	cacheValue, err := json.Marshal(result)
	if err != nil {
		log.Printf("Failed to marshal result for caching: %v", err)
	} else {
		if err := redisClient.Set(ctx, cacheKey, cacheValue, time.Hour).Err(); err != nil {
			log.Printf("Failed to cache result for %s in Redis: %v", job.ImageName, err)
		}
	}
	result["ImageName"] = job.ImageName
	result["ResultType"] = "Processed"
	return result
}

func writeResultsToFile(results DetectionResults) {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	output, _ := json.Marshal(results)
	outputStr := string(output)

	if results["ResultType"] == "Processed" {
		face_detection_Results.Processed++
	} else if results["ResultType"] == "Cached" {
		face_detection_Results.Cached++
	}
	face_detection_Results.TotalImages++
	face_detection_Results.Faces = append(face_detection_Results.Faces, results)

	if _, err := resultsFile.WriteString(outputStr + "\n"); err != nil {
		log.Println("Failed to write results to the file:", err)
	}
}

func handleError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{"error": message})
}
