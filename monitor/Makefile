BIN := liims-monitor
SRC := go.mod $(wildcard *.go) index.html
LDFLAGS := -s -w

.PHONY: all clean

all: $(BIN)

$(BIN): $(SRC)
	go build -ldflags="$(LDFLAGS)"

clean:
	rm -f $(BIN)
