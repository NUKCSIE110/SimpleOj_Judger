package main

import (
	"crypto/md5"
	"fmt"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"time"
)

type Submission struct {
	uuid   string
	qid    int
	code   string
	lang   int
	result string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

var baseDir *string

func init() {
	baseDir = flag.String("d", ".", "Base directory")
	flag.Parse();
	log.Printf("Workdir: %s\n", *baseDir);
}

func main() {
	readyQueue := make(chan Submission)
	judgedQueue := make(chan Submission)
	go Judger(readyQueue, judgedQueue)
	go Emmiter(judgedQueue)
	handleHTTP := makeHTTP(readyQueue)

	http.HandleFunc("/", handleHTTP)
	http.ListenAndServe(":4321", nil)
}

func Judger(src chan Submission, dist chan Submission) {
	for {
		s := <-src
		fileName := *baseDir + "/code/" + s.uuid
		var compilerName string
		var standard string
		if s.lang == 0 {
			fileName += ".c"
			compilerName = "gcc"
			standard = "c11"
		} else {
			fileName += ".cpp"
			compilerName = "g++"
			standard = "c++11"
		}

		err := ioutil.WriteFile(fileName, []byte(s.code), 0644)
		check(err)
		log.Printf("[%s] File wrote\n", s.uuid[:7])

		log.Printf("[%s] Compiling...\n", s.uuid[:7])
		compiler := exec.Command(compilerName, "-O2", "-std="+standard, "-static", fileName, "-o", "code/"+s.uuid)
		err = compiler.Run()
		if err != nil {
			s.result = "CE"
			dist <- s
			continue
		}

		log.Printf("[%s] Judging...\n", s.uuid[:7])
		done := make(chan string)
		cpid := make(chan int)
		go JudgeThread(done, cpid, s)
		pid := <-cpid
		log.Printf("[%s] PID: %d\n", s.uuid[:7], pid)
		timer1 := time.NewTimer(time.Second * 5)
		select {
		case result := <-done:
			s.result = result
			dist <- s
		case <-timer1.C:
			s.result = "TLE"
			proc, err := os.FindProcess(pid)
			check(err)
			err = proc.Kill()
			check(err)
			log.Printf("[%s] Killed %d\n", s.uuid[:7], pid)
			dist <- s
		}
		os.Remove("code/" + s.uuid)
	}
}

func JudgeThread(done chan string, pid chan int, s Submission) {
	cmd := exec.Command(*baseDir + "/code/" + s.uuid)
	stdin, err := cmd.StdinPipe()
	check(err)
	in_buf, err := ioutil.ReadFile(fmt.Sprintf("%s/testData/p%d.in", *baseDir, s.qid))
	check(err)
	stdout, err := cmd.StdoutPipe()
	check(err)
	defer stdout.Close()
	ac_buf, err := ioutil.ReadFile(fmt.Sprintf("%s/testData/p%d.out", *baseDir, s.qid))
	check(err)
	err = cmd.Start()
	if err != nil {
		done <- "RE"
		return
	}
	pid <- cmd.Process.Pid
	go func() {
		defer stdin.Close()
		io.WriteString(stdin, string(in_buf))
	}()
	outBytes, err := ioutil.ReadAll(stdout)
	check(err)
	if md5.Sum(outBytes) == md5.Sum(ac_buf) {
		done <- "AC"
	} else {
		done <- "WA"
	}
}

func Emmiter(src chan Submission) {
	for {
		select {
		case s := <-src:
			log.Printf("[%s] Result: %s\n", s.uuid[:7], s.result)
			resp, _ := http.PostForm(fmt.Sprintf("http://0.0.0.0:3000/judge/%s/update", s.uuid), url.Values{"result": {s.result}})
			log.Printf("[%s] Emmited: %s\n", s.uuid[:7], resp.Status)
		}
	}
}

func makeHTTP(readyQueue chan Submission) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		uuid := r.Form.Get("uuid")
		code := r.Form.Get("code")
		qid, _ := strconv.Atoi(r.Form.Get("qid"))
		lang, _ := strconv.Atoi(r.Form.Get("lang"))
		s := Submission{uuid: uuid, code: code, qid: qid, lang: lang}
		readyQueue <- s
		log.Printf("[New] %s\n", uuid[:7])
		fmt.Fprint(w, uuid)
	}
}
