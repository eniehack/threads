.PHONY: threads threadsctl clean all
PROGRAM = threads threadsctl

all: threadsctl threads

threadsctl:
	go build -o threadsctl ./cmd/ctl/main.go

threads:
	go build -o threads ./cmd/server/main.go

clean:
	${RM} ${PROGRAM}
