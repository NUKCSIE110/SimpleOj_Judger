package main

import (
    "fmt"
    "net/http"
    "strconv"
    "log"
    "net/url"
)

type Submission struct{
    uuid string
    qid int
    code string
    lang int
    result string
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

func Judger(src chan Submission, dist chan Submission){
    for{
        select{
        case s := <-src:
            log.Printf("[%s] Judging...\n",s.uuid[:7])
            s.result = "AC"
            log.Printf("[%s] Result: %s\n", s.uuid[:7], s.result)
            dist <- s
        }

    }
}

func Emmiter(src chan Submission){
    for{
        select{
        case s:= <-src:
            resp, _ := http.PostForm(fmt.Sprintf("http://0.0.0.0:3000/judge/%s/update",s.uuid),url.Values{"result": {s.result}})
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
        s := Submission{uuid:uuid, code:code, qid:qid, lang:lang}
        readyQueue <- s
        log.Printf("[New] %s\n", uuid[:7])
        fmt.Fprint(w, uuid)
    }
}
