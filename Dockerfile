FROM golang

WORKDIR /go/src/github.com/NUKCSIE110/SimpleOj_Judger
COPY . /go/src/github.com/NUKCSIE110/SimpleOj_Judger

#RUN go-wrapper download github.com/NUKCSIE110/SimpleOj_Judger
RUN go-wrapper install github.com/NUKCSIE110/SimpleOj_Judger
RUN mkdir /var/Judger
RUN mkdir /var/Judger/code
RUN mkdir /var/Judger/testData
RUN apt-get update
RUN yes | apt-get install build-essential

VOLUME "/var/Judger"
EXPOSE 4321
ENTRYPOINT ["/go/bin/SimpleOj_Judger", "-d", "/var/Judger"]
