package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

func ExtraTar(filename string) bool {
	log.Println("ExtraTar++ " + filename)
	tarfile := "/Users/qa/go/" + filename + ".tar"
	if _, err := os.Stat(tarfile); os.IsNotExist(err) {
		// path does not exist
		return false
	}
	//create folder
	logfolder := "/Users/qa/go/" + filename + ".logarchive"
	if os.Mkdir(logfolder, 0755) != nil {
		return false
	}
	//tar -xvf tarfile.dat -C aaa.logarchive
	cmd := exec.Command("tar", "-xvf", tarfile, "-C", logfolder)
	// Get a pipe to read from standard out
	r, _ := cmd.StdoutPipe()

	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to ensure we get all output
	done := make(chan bool)

	// Create a scanner which scans r in a line-by-line fashion
	scanner := bufio.NewScanner(r)
	// Use the scanner to scan the output line by line and log it
	go func() {

		// Read line by line and process it
		for scanner.Scan() {
			//line := scanner.Text()
			//fmt.Println(line)
		}
		// We're all done, unblock the channel
		done <- true

	}()

	// Start the command and check for errors
	cmd.Start()

	// Wait for all output to be processed
	<-done

	// Wait for the command to finish
	err := cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			log.Printf("ExitCode=%d\n", waitStatus.ExitStatus())
			return false
		}
	} else {
		// Success
		waitStatus := cmd.ProcessState.Sys().(syscall.WaitStatus)
		log.Printf("ExitCode=%d\n", waitStatus.ExitStatus())
	}
	log.Println("ExtraTar-- ")
	return true
}

func findMaxCapacity(filename string) (string, error) {
	//log show --archive aaa.logarchive --start "2021-10-04" --process powerd  ｜ grep MaxCapacity
	tarfile := "/Users/qa/go/" + filename + ".logarchive"
	startdate := time.Now().AddDate(0, 0, -7).Format("2006-01-02")
	//scmd := fmt.Sprintf("log show --archive %s --start %s --process powerd ", tarfile, startdate)
	//cmd := exec.Command("bash", "-c", scmd)
	cmd := exec.Command("log", "show", "--archive", tarfile, "--start", startdate, "--process", "powerd")
	// Get a pipe to read from standard out
	r, _ := cmd.StdoutPipe()

	// Use the same pipe for standard error
	cmd.Stderr = cmd.Stdout

	// Make a new channel which will be used to ensure we get all output
	done := make(chan bool)

	maxcap := "0"
	// Create a scanner which scans r in a line-by-line fashion
	scanner := bufio.NewScanner(r)

	//Updated Battery Health: Flags:34078723 State:5 MaxCapacity:100 CycleCount:(null)
	reg := regexp.MustCompile(`Updated Battery Health: Flags:\d+ State:\d+ MaxCapacity:(\d+) CycleCount:`)
	// Use the scanner to scan the output line by line and log it
	go func() {

		// Read line by line and process it
		for scanner.Scan() {
			line := scanner.Text()
			vs := reg.FindStringSubmatch(line)
			if len(vs) == 2 {
				maxcap = vs[1]
			}

			//fmt.Println(line)
		}
		// We're all done, unblock the channel
		done <- true

	}()

	// Start the command and check for errors
	cmd.Start()

	// Wait for all output to be processed
	<-done

	// Wait for the command to finish
	err := cmd.Wait()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			waitStatus := exitError.Sys().(syscall.WaitStatus)
			log.Printf("ExitCode=%d\n", waitStatus.ExitStatus())
			return maxcap, err
		}
	} else {
		// Success
		waitStatus := cmd.ProcessState.Sys().(syscall.WaitStatus)
		log.Printf("ExitCode=%d\n", waitStatus.ExitStatus())
	}

	return maxcap, nil
}

func CleanSpace(filename string) {
	//delete tar
	//delete tar extra folder
	tarfile := "/Users/qa/go/" + filename + ".tar"
	if _, err := os.Stat(tarfile); !os.IsNotExist(err) {
		// path does  exist
		os.Remove(tarfile)
	}
	//create folder
	logfolder := "/Users/qa/go/" + filename + ".logarchive"
	if _, err := os.Stat(logfolder); !os.IsNotExist(err) {
		// path does  exist
		os.RemoveAll(logfolder)
	}
}

func setupRouter() *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	r := gin.Default()

	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"version": "1.0.0.0", // cast it to string before showing
			"author":  "Jeffery Zhang",
			"company": "FutureDial",
		})
	})

	// Get user value
	r.GET("/v1/:id", func(c *gin.Context) {
		tarname := c.Params.ByName("id")
		fmt.Println(tarname)
		if ExtraTar(tarname) {
			mc, err := findMaxCapacity(tarname)
			CleanSpace(tarname)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":        http.StatusInternalServerError,
					"MaxCapacity": "0", // cast it to string before showing
					"error":       err,
				})
			}
			c.JSON(http.StatusOK, gin.H{
				"code":        http.StatusOK,
				"MaxCapacity": mc, // cast it to string before showing
			})
		} else {
			c.JSON(http.StatusLocked, gin.H{
				"code":  http.StatusLocked,
				"error": "error file extra failed", // cast it to string before showing
			})
		}

	})

	return r
}

func main() {
	//start bonjour :dns-sd -R "traceLogs" _http._tcp . 8080 path=/version

	router := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	srv := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 5 seconds.
	quit := make(chan os.Signal, 1)
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be catch, so don't need add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
