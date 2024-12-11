FROM golang:latest
RUN mkdir /OpenElect
WORKDIR /OpenElect
COPY . .
RUN go mod download
EXPOSE 8080
CMD ["go", "run", "."]
