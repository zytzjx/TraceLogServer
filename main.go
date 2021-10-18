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
	"path/filepath"
	"regexp"
	"runtime"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/grandcat/zeroconf"
)

const (
	defaultpath string = "/Users/qa/go/"
	VERSION     string = "1.0.0.2"
	SERVERPORT  int    = 8080
)

func ExtraTar(path, filename string) bool {
	log.Println("ExtraTar++ " + filename)
	tarfile := filepath.Join("/", path, filename+".tar")
	if _, err := os.Stat(tarfile); os.IsNotExist(err) {
		// path does not exist
		return false
	}
	//create folder
	logfolder := filepath.Join("/", path, filename+".logarchive")
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

func CopyFileFromNetDiver(path, filename string) (string, error) {
	// cp -R /Volumes/go/1.logarchive /Users/qa/go
	log.Printf("CopyFileFromNetDiver++ %s\n", filename)
	logpath := filepath.Join("/", path, filename+".logarchive")
	log.Println(logpath)
	cmd := exec.Command("cp", "-R", logpath, defaultpath)
	err := cmd.Run()
	if err != nil {
		return "", err
	}
	log.Printf("CopyFileFromNetDiver-- %s\n", filename)
	return defaultpath, nil
}

func findMaxCapacity(path, filename string) (string, string, error) {
	//log show --archive aaa.logarchive --start "2021-10-04" --process powerd  ｜ grep MaxCapacity
	//"/Users/qa/go/"
	tarfile := filepath.Join("/", path, filename+".logarchive")
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
	var linecap string
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
				linecap = line
			}
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
			return maxcap, linecap, err
		}
	} else {
		// Success
		waitStatus := cmd.ProcessState.Sys().(syscall.WaitStatus)
		log.Printf("ExitCode=%d\n", waitStatus.ExitStatus())
	}

	return maxcap, linecap, nil
}

func CleanSpace(path, filename string) {
	//delete tar
	//delete tar extra folder
	tarfile := filepath.Join("/", path, filename+".tar")
	if _, err := os.Stat(tarfile); !os.IsNotExist(err) {
		// path does  exist
		os.Remove(tarfile)
	}
	//create folder
	logfolder := filepath.Join("/", path, filename+".logarchive")
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
		path := c.DefaultQuery("path", defaultpath)
		fmt.Println(path)
		c.JSON(http.StatusOK, gin.H{
			"code":    http.StatusOK,
			"version": VERSION, // cast it to string before showing
			"author":  "Jeffery Zhang",
			"company": "FutureDial",
			"ip":      c.ClientIP(),
		})
	})

	r.GET("v3/:id", func(c *gin.Context) {
		tarname := c.Params.ByName("id")
		path := c.DefaultQuery("path", defaultpath)
		log.Println(path)
		curpath, err := CopyFileFromNetDiver(path, tarname)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":        http.StatusInternalServerError,
				"MaxCapacity": "0", // cast it to string before showing
				"error":       err,
				"info":        "Copy files failed",
			})
			return
		}
		mc, line, err := findMaxCapacity(curpath, tarname)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":        http.StatusInternalServerError,
				"MaxCapacity": "0", // cast it to string before showing
				"error":       err,
			})
			return
		}
		log.Printf("[%s]MaxCapacity=%s", tarname, mc)
		c.JSON(http.StatusOK, gin.H{
			"code":        http.StatusOK,
			"MaxCapacity": mc, // cast it to string before showing
			"line":        line,
		})

	})
	r.GET("/v2/:id", func(c *gin.Context) {
		tarname := c.Params.ByName("id")
		path := c.DefaultQuery("path", defaultpath)
		log.Println(path)
		mc, line, err := findMaxCapacity(path, tarname)
		//CleanSpace(path, tarname)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":        http.StatusInternalServerError,
				"MaxCapacity": "0", // cast it to string before showing
				"error":       err,
			})
			return
		}
		log.Printf("[%s]MaxCapacity=%s", tarname, mc)
		c.JSON(http.StatusOK, gin.H{
			"code":        http.StatusOK,
			"MaxCapacity": mc, // cast it to string before showing
			"line":        line,
		})
	})
	// Get user value
	r.GET("/v1/:id", func(c *gin.Context) {
		tarname := c.Params.ByName("id")
		fmt.Println(tarname)
		if ExtraTar(defaultpath, tarname) {
			mc, line, err := findMaxCapacity(defaultpath, tarname)
			CleanSpace(defaultpath, tarname)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"code":        http.StatusInternalServerError,
					"MaxCapacity": "0", // cast it to string before showing
					"error":       err,
				})
				return
			}
			log.Printf("[%s]MaxCapacity=%s", tarname, mc)
			c.JSON(http.StatusOK, gin.H{
				"code":        http.StatusOK,
				"MaxCapacity": mc, // cast it to string before showing
				"line":        line,
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
	runtime.GOMAXPROCS(runtime.NumCPU())

	server, err := zeroconf.Register("TRACELOG", "_tracelog._tcp", "local.", SERVERPORT, []string{"version=" + VERSION, "author=jeffery"}, nil)
	if err != nil {
		panic(err)
	}
	defer server.Shutdown()

	router := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", SERVERPORT),
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
