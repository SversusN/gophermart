FROM golang:1.22-alpine as builder

WORKDIR /usr/local/src

RUN apk --no-cach add bash git make gcc gettext musl-dev

#dependencies
COPY ["./go.mod","./go.sum", "./" ]
RUN go mod download

#build
COPY ./ ./
RUN go build -o ./bin/app cmd/gophermart/main.go

FROM alpine as runner

COPY --from=builder /usr/local/src/bin/app /
COPY ./cmd/accrual/accrual_linux_amd64 /
CMD ["./app", "./accrual_linux_amd64"]
EXPOSE 8080