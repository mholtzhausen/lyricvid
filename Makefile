BINARY := lyricvid
BUILD_DIR := build

.PHONY: build clean

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY) .

clean:
	rm -rf $(BUILD_DIR)
